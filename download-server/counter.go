package main

import (
	"fmt"
	"sync/atomic"
)

func NewCounter(name string) Counter {
	return Counter{name: name}
}

type Counter struct {
	val uint64
	name string
}

func (c *Counter) Increment() uint64 {
	return atomic.AddUint64(&c.val, 1)
}

func (c *Counter) Value() uint64 {
	return atomic.LoadUint64(&c.val)
}

func (c Counter) String() string {
	return fmt.Sprintf("%s = %d", c.name, c.Value())
}