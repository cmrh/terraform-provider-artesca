package client

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLockBucketSerializesSameBucket(t *testing.T) {
	c := NewS3Client("http://unused", "us-east-1", false)
	var inside, maxInside int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := c.LockBucket("b")
			defer unlock()
			n := atomic.AddInt32(&inside, 1)
			for {
				m := atomic.LoadInt32(&maxInside)
				if n <= m || atomic.CompareAndSwapInt32(&maxInside, m, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			atomic.AddInt32(&inside, -1)
		}()
	}
	wg.Wait()
	if maxInside != 1 {
		t.Errorf("max concurrent holders of one bucket's lock = %d, want 1", maxInside)
	}
}

func TestLockBucketIndependentBuckets(t *testing.T) {
	c := NewS3Client("http://unused", "us-east-1", false)
	unlockA := c.LockBucket("a")
	defer unlockA()

	done := make(chan struct{})
	go func() {
		unlockB := c.LockBucket("b")
		unlockB()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("locking bucket b blocked while bucket a was locked")
	}
}
