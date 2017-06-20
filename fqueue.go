package fqueue

import "errors"

type cache struct {
	val interface{}
	// readable
	flag bool
}
type queue struct {
	_padding0 [8]uint64
	mask      uint32
	_padding1 [8]uint64
	buff      []cache
	_padding2 [8]uint64
	cons      uint32
	_padding3 [8]uint64
	prod      uint32
	_padding4 [8]uint64
}

var ErrIllegalCap = errors.New("fqueue: the cap must be 1 ~ 2^32")

// cap must be 2^n
// 队列的大小建议超过 1s 的"瞬间"新增消息或者读取消息的数量
func New(n uint8) (q *queue, err error) {
	if n > 32 || n == 0 {
		err = ErrIllegalCap
		return
	}
	q = new(queue)
	q.mask = (1 << n) - 1
	q.buff = make([]cache, 1<<n)
	return
}

const (
	active_spin     = 3
	active_spin_cnt = 10
)

//go:nosplit
func doSpin() {
	spin(active_spin_cnt)
}

//go:noescape
func spin(cycles uint32)

func casUint32(addr *uint32, old, new uint32) (swapped bool)

func (q *queue) Push(v interface{}) (ok bool) {

	mask := q.mask

	// first try
	cons := q.cons
	prod := q.prod
	// if cons catch prod
	var prodCnt uint32
	if prod >= cons {
		prodCnt = prod - cons
	} else {
		prodCnt = mask + prod - cons
	}
	if prodCnt >= mask {
		//runtime.Gosched()
		return false
	}
	prodNew := prod + 1
	if casUint32(&q.prod, prod, prodNew) {
		cache := &q.buff[prodNew&mask]
		if !cache.flag {
			cache.val = v
			cache.flag = true
			//runtime.Gosched()
			return true
		} else {
			// 不太可能会到这里，前面已经判断过 cons catch prod
			//runtime.Gosched()
			return false
		}

	}

	// spin try
	iter := 0
	for iter < active_spin {
		cons = q.cons
		prod = q.prod

		if prod >= cons {
			prodCnt = prod - cons
		} else {
			prodCnt = mask + prod - cons
		}
		if prodCnt >= mask {
			//runtime.Gosched()
			return false
		}

		prodNew = prod + 1
		if casUint32(&q.prod, prod, prodNew) {
			cache := &q.buff[prodNew&mask]
			if !cache.flag {
				cache.val = v
				cache.flag = true
				//runtime.Gosched()
				return true
			} else {
				//runtime.Gosched()
				return false
			}
		} else {
			doSpin()
			iter++
			continue
		}
	}
	// TODO 是否需要出让时间片
	//runtime.Gosched()
	return false
}

func (q *queue) Get() (ok bool, v interface{}) {

	mask := q.mask

	// first try
	prod := q.prod
	cons := q.cons
	var prodCnt uint32
	// if prod catch cons
	if prod >= cons {
		prodCnt = prod - cons
	} else {
		prodCnt = mask + prod - cons
	}

	if prodCnt < 1 {
		//runtime.Gosched()
		return false, nil
	}

	consNew := cons + 1
	if casUint32(&q.cons, cons, consNew) {
		cache := &q.buff[consNew&mask]
		if cache.flag {
			v = cache.val
			cache.val = nil
			cache.flag = false
			//runtime.Gosched()
			return true, v
		} else {
			//runtime.Gosched()
			return false, nil
		}
	}

	iter := 0
	for iter < active_spin {
		prod = q.prod
		cons = q.cons
		if prod >= cons {
			prodCnt = prod - cons
		} else {
			prodCnt = mask + prod - cons
		}
		if prodCnt < 1 {
			//runtime.Gosched()
			return false, nil
		}
		consNew := cons + 1
		if casUint32(&q.cons, cons, consNew) {
			cache := &q.buff[consNew&mask]
			if cache.flag {
				v = cache.val
				cache.val = nil
				cache.flag = false
				//runtime.Gosched()
				return
			} else {
				//runtime.Gosched()
				return false, nil
			}
		} else {
			doSpin()
			iter++
			continue
		}
	}
	//runtime.Gosched()
	return false, nil
}
