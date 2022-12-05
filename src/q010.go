/*
@Time : 2021/1/29 上午9:31
@Author : sunyujia
@File : q010
@Software: GoLand
*/
package main

import (
	"log"
	"sync"
	"time"
)

type sp interface {
	Out(key string, val interface{})                  //存入key /val，如果该key读取的goroutine挂起，则唤醒。此方法不会阻塞，时刻都可以立即执行并返回
	Rd(key string, timeout time.Duration) interface{} //读取一个key，如果key不存在阻塞，等待key存在或者超时
}

type Map struct {
	c   map[string]*entry
	rmx *sync.RWMutex
}
type entry struct {
	ch      chan struct{}
	value   interface{}
	isExist bool
}

func (m *Map) Out(key string, val interface{}) {
	m.rmx.Lock()
	defer m.rmx.Unlock()
	item, ok := m.c[key]
	if !ok {
		// 不存在时候，直接设置value
		m.c[key] = &entry{
			value:   val,
			isExist: true,
		}
		return
	}
	// 存在时候，先更新value
	item.value = val
	if !item.isExist {
		// 如果isExist为false,则表示有读的请求
		if item.ch != nil {
			// 关闭ch,读ch的就可以退出循环了
			close(item.ch)
			item.ch = nil
		}
	}
	return
}

func (m *Map) Rd(key string, timeout time.Duration) interface{} {
	m.rmx.RLock()
	value, keyValueExist := m.c[key]
	m.rmx.RUnlock()
	if keyValueExist && value.isExist {
		return value.value
	}
	if !keyValueExist {
		m.rmx.Lock()
		value = &entry{ch: make(chan struct{}), isExist: false}
		m.c[key] = value
		m.rmx.Unlock()
	}
	log.Println("协程阻塞 -> ", key)
	select {
	case <-value.ch:
		return value.value
	case <-time.After(timeout):
		log.Println("协程超时 -> ", key)
		return nil
	}

}

func main() {
	mapval := Map{
		c:   make(map[string]*entry),
		rmx: &sync.RWMutex{},
	}

	for i := 0; i < 10; i++ {
		go func() {
			val := mapval.Rd("key", time.Second*6)
			log.Println("读取值为->", val)
		}()
	}

	time.Sleep(time.Second * 3)
	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			mapval.Out("key", val)
		}(i)
	}
	wg.Wait()
	time.Sleep(time.Second)
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
