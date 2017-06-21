package fqueue

import "errors"

type cache struct {
	val interface{}
	// if readable
	flag bool
}
type queue struct {
	mask      uint32
	buff      []cache
	_padding2 [8]uint64
	cons      uint32
	_padding3 [8]uint64
	prod      uint32
	_padding4 [8]uint64
}

var ErrIllegalCap = errors.New("fqueue: the cap must be 1 ~ 2^32")

// cap must be 2^n
// advise: the cap of queue should > num of push/get msg in 1s
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
	// spins loops
	spins = 10
	// nop ops in CPU
	spin_cnt = 300
)

// status
const (
	success     uint8 = 0
	catchCons   uint8 = 1
	catchProd   uint8 = 1
	prodTooFast uint8 = 2
	consTooFast uint8 = 2
	spinOut     uint8 = 3
)

// push a new msg into queue
func (q *queue) Push(msg interface{}) (statusCode uint8) {

	mask := q.mask
	iter := 0
	for iter < spins {
		cons := q.cons
		prod := q.prod

		// if catch cons
		var prodCnt uint32
		if prod >= cons {
			prodCnt = prod - cons
		} else {
			prodCnt = mask + prod - cons
		}
		if prodCnt >= mask {
			return catchCons
		}

		prodNew := prod + 1
		if casUint32(&q.prod, prod, prodNew) {
			cache := &q.buff[prodNew&mask]
			if !cache.flag {
				cache.val = msg
				cache.flag = true
				return success
			} else {
				return prodTooFast
			}
		} else {
			doSpin()
			iter++
			continue
		}
	}
	return spinOut
}

func (q *queue) Get() (statusCode uint8, v interface{}) {

	mask := q.mask

	iter := 0
	for iter < spins {
		prod := q.prod
		cons := q.cons
		var prodCnt uint32
		if prod >= cons {
			prodCnt = prod - cons
		} else {
			prodCnt = mask + prod - cons
		}
		if prodCnt < 1 {
			return catchProd, nil
		}
		consNew := cons + 1
		if casUint32(&q.cons, cons, consNew) {
			cache := &q.buff[consNew&mask]
			if cache.flag {
				v = cache.val
				cache.val = nil
				cache.flag = false
				return success, v
			} else {
				return consTooFast, nil
			}
		} else {
			doSpin()
			iter++
			continue
		}
	}
	return spinOut, nil
}

//go:nosplit
func doSpin() {
	spin(spin_cnt)
}

//go:noescape
func spin(cycles uint32)

func casUint32(addr *uint32, old, new uint32) (swapped bool)
