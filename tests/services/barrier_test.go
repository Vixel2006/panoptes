package service_test

import (
	"sync"
	"testing"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/services"
)

func TestBarrierBlocksUntilRelease(t *testing.T) {
	b := service.NewBarrier()

	started := make(chan struct{})
	unblocked := make(chan struct{})

	go func() {
		close(started)
		b.Lock()
		close(unblocked)
		b.Unlock()
	}()

	<-started
	// small window for the goroutine to reach Lock() and block on hold channel
	time.Sleep(5 * time.Millisecond)

	b.Release(true)
	select {
	case <-unblocked:
	case <-time.After(time.Second):
		t.Fatal("Lock did not unblock after Release")
	}
}

func TestBarrierReturnsReleasedDecision(t *testing.T) {
	b := service.NewBarrier()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		b.Lock()
		b.Unlock()
		wg.Done()
	}()

	time.Sleep(5 * time.Millisecond)
	b.Release(false)
	wg.Wait()

	// decision should be false from the Release
	if got := b.Decision(); got != false {
		t.Errorf("Decision() = %v, want false", got)
	}
}

func TestBarrierInactivePassesThrough(t *testing.T) {
	b := service.NewBarrier()

	b.SetActive(false)

	done := make(chan struct{})
	go func() {
		b.Lock()
		close(done)
		b.Unlock()
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Lock blocked when barrier was inactive")
	}

	if !b.Decision() {
		t.Error("expected Decision() = true when inactive")
	}
}

func TestBarrierReactivationBlocksAgain(t *testing.T) {
	b := service.NewBarrier()

	b.SetActive(false)
	// Let a goroutine through while inactive
	done1 := make(chan struct{})
	go func() {
		b.Lock()
		close(done1)
		b.Unlock()
	}()
	<-done1

	// Reactivate
	b.SetActive(true)

	// Now Lock should block again
	blocked := make(chan struct{})
	go func() {
		b.Lock()
		close(blocked)
		b.Unlock()
	}()

	select {
	case <-blocked:
		t.Fatal("Lock should have blocked after reactivation")
	case <-time.After(10 * time.Millisecond):
	}

	b.Release(true)
	select {
	case <-blocked:
	case <-time.After(time.Second):
		t.Fatal("Lock did not unblock after Release")
	}
}
