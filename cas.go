package atomic

import (
	"math"
	"sync/atomic"
)

type atomicFloatCAS struct{ u64 uint64 }

func NewAtomicFloatCAS(initial float64) *atomicFloatCAS {
	return &atomicFloatCAS{u64: math.Float64bits(initial)}
}

// Add attempts to add delta to the value stored in the atomic float and return
// the new value.
func (a *atomicFloatCAS) Add(delta float64) float64 {
	var newValue float64
	var oldBits, newBits uint64
	for {
		oldBits = atomic.LoadUint64(&a.u64)
		newValue = math.Float64frombits(oldBits) + delta
		newBits = math.Float64bits(newValue)
		if atomic.CompareAndSwapUint64(&a.u64, oldBits, newBits) {
			return newValue
		}
	}
}

// Load atomically loads the current atomic float value.
func (a *atomicFloatCAS) Load() float64 {
	return math.Float64frombits(atomic.LoadUint64(&a.u64))
}

// Store atomically stores new into the atomic float.
func (a *atomicFloatCAS) Store(new float64) {
	atomic.StoreUint64(&a.u64, math.Float64bits(new))
}

// Swap atomically stores new and returns the previous value.
func (a *atomicFloatCAS) Swap(new float64) float64 {
	return math.Float64frombits(atomic.SwapUint64(&a.u64, math.Float64bits(new)))
}
