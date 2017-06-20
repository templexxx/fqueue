package fqueue

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCompelete(t *testing.T) {
	q, err := New(6)
	if err != nil {
		t.Fatal(err)
	}
	worker := 2
	// go push
	wg := &sync.WaitGroup{}
	wg.Add(worker)
	var pushCnt uint32
	for i := 1; i <= worker; i++ {
		go push(wg, q, &pushCnt)
	}

	// go Get
	time.Sleep(100 * time.Microsecond)
	wg1 := &sync.WaitGroup{}
	wg1.Add(worker)
	var getCnt uint32
	for i := 0; i < worker; i++ {
		go get(wg1, q, &getCnt)
	}
	wg.Wait()
	wg1.Wait()
	if pushCnt != 63 {
		t.Fatalf("push miss; expect: 63, got: %d\n", pushCnt)
	}
	if getCnt != 63 {
		t.Fatalf("get miss; expect: 63, got: %d\n", getCnt)
	}
}

func push(wg *sync.WaitGroup, q *queue, pushCnt *uint32) {

	defer wg.Done()
	for i := 0; i < 32; i++ {
		ok := q.Push(0)
		if ok {
			atomic.AddUint32(pushCnt, 1)
		}
	}
}

func get(wg *sync.WaitGroup, q *queue, getCnt *uint32) {
	defer wg.Done()
	for i := 0; i < 32; i++ {
		ok, _ := q.Get()
		if ok {
			atomic.AddUint32(getCnt, 1)
		}
	}
}

func TestPerfFQueuePushGet(t *testing.T) {
	numMsg := 1 << 16
	worker := runtime.NumCPU() * 4
	numMsgPerGo := numMsg / worker
	var putOK, getOK uint32
	putD, getD := testQueuePutGet(t, worker, numMsgPerGo, &getOK, &putOK)
	t.Logf("Put: %d, use: %v, %v/op, ok: %v", numMsg, putD, putD/time.Duration(numMsg), putOK)
	t.Logf("Get: %d, use: %v, %v/op, ok: %v", numMsg, getD, getD/time.Duration(numMsg), getOK)
}

func testQueuePutGet(t *testing.T, worker, cnt int, getOK, putOK *uint32) (put time.Duration, get time.Duration) {
	var wg sync.WaitGroup
	wg.Add(worker)
	q, err := New(uint8(worker))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	for i := 0; i < worker; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < cnt; j++ {
				ok := q.Push(0)
				if ok {
					atomic.AddUint32(putOK, 1)
				}
			}
		}()
	}
	wg.Wait()
	end := time.Now()
	put = end.Sub(start)

	wg.Add(worker)
	start = time.Now()
	for i := 0; i < worker; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < cnt; j++ {
				ok, _ := q.Get()
				if ok {
					atomic.AddUint32(getOK, 1)
				}
			}
		}()
	}
	wg.Wait()
	end = time.Now()
	get = end.Sub(start)
	return put, get
}
