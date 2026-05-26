package service

import (
	"sync"
)

type Barrier struct {
	sync.Mutex

	active   bool
	hold     chan bool
	decision bool
}

func NewBarrier() *Barrier {
	return &Barrier{
		active:   true,
		hold:     make(chan bool, 1),
		decision: true,
	}
}

func (b *Barrier) Lock() {
	b.Mutex.Lock()
	if b.active {
		b.Mutex.Unlock()
		b.decision = <-b.hold
		b.Mutex.Lock()
	} else {
		b.decision = true
	}
}

func (b *Barrier) Unlock() {
	b.Mutex.Unlock()
}

func (b *Barrier) Decision() bool {
	return b.decision
}

func (b *Barrier) Release(forward bool) {
	b.hold <- forward
}

func (b *Barrier) SetActive(active bool) {
	b.Mutex.Lock()
	b.active = active
	if !active {
		select {
		case b.hold <- true:
		default:
		}
	}
	b.Mutex.Unlock()
}
