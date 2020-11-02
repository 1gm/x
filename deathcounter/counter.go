package main

import (
	"sync/atomic"
)

type counter struct {
	v int64
}

func (c *counter) Increment() int64 {
	return atomic.AddInt64(&c.v, 1)
}

func (c *counter) Decrement() int64 {
	return atomic.AddInt64(&c.v, -1)
}

func newCounter(start int64) *counter {
	return &counter{
		v: start,
	}
}
