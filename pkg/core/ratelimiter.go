package core

import (
	"errors"
	"sync"
	"time"
)

type RateLimiter struct {
	bandwidth      uint
	tokens         uint64
	capacity       uint64
	ticker         *time.Ticker
	mutex          sync.Mutex
	mutexWriteWait sync.Mutex
	mutexWait      sync.Mutex
	done           sync.WaitGroup
	waiting        uint
}

func NewRateLimiter(bandwidth uint) *RateLimiter {
	rl := &RateLimiter{
		bandwidth: bandwidth,
		tokens:    0,
		ticker:    time.NewTicker(time.Millisecond),
		capacity:  uint64(bandwidth << 4),
		waiting:   0,
	}
	go rl.Run()
	return rl
}

func (rateLimiter *RateLimiter) Run() {
	for {
		<-rateLimiter.ticker.C
		rateLimiter.add(rateLimiter.bandwidth)
	}
}

func (rateLimiter *RateLimiter) add(tokens uint) {
	rateLimiter.mutex.Lock()
	defer rateLimiter.mutex.Unlock()
	result := rateLimiter.tokens + uint64(tokens)
	if result < rateLimiter.capacity {
		rateLimiter.tokens = result
	} else {
		rateLimiter.tokens = rateLimiter.capacity
	}
	rateLimiter.mutexWriteWait.Lock()
	defer rateLimiter.mutexWriteWait.Unlock()
	if rateLimiter.waiting != 0 && rateLimiter.tokens >= uint64(rateLimiter.waiting) {
		if rateLimiter.tokens > uint64(rateLimiter.waiting) {
			rateLimiter.tokens -= uint64(rateLimiter.waiting)
		} else {
			rateLimiter.tokens = 0
		}
		rateLimiter.waiting = 0
		rateLimiter.done.Done()
	}
}

func (rateLimiter *RateLimiter) Wait(tokens uint) error {
	rateLimiter.mutexWait.Lock()
	defer rateLimiter.mutexWait.Unlock()
	if uint64(tokens) > rateLimiter.capacity {
		return errors.New("out of capacity")
	}
	rateLimiter.mutexWriteWait.Lock()
	rateLimiter.waiting = tokens
	rateLimiter.done.Add(1)
	rateLimiter.mutexWriteWait.Unlock()
	rateLimiter.done.Wait()
	return nil
}

func (rateLimiter *RateLimiter) Available() uint64 {
	return rateLimiter.tokens
}

func (rateLimiter *RateLimiter) Capacity() uint64 {
	return rateLimiter.capacity
}
