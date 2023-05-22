package demo

import (
	"fmt"
	"os"
	"sync"
)

type ConcurrentMap[K comparable, V any] struct {
	m sync.Map
}

func (c *ConcurrentMap[K, V]) Load(key K) (value V, ok bool) {
	v, ok := c.m.Load(key)
	if ok {
		return v.(V), true
	}
	return
}

func (c *ConcurrentMap[K, V]) Store(key K, value V) {
	c.m.Store(key, value)
}

func (c *ConcurrentMap[K, V]) Delete(key K) {
	c.m.Delete(key)
}

func (c *ConcurrentMap[K, V]) Range(f func(k K, v V) bool) {
	c.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func Stderr(str string) {
	_, err := os.Stderr.Write([]byte(str + "\n"))
	if err != nil {
		fmt.Println("std err failed", err, "origin:", str)
	}
}
