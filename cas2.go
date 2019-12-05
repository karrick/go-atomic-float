package atomic

import (
	"math"
	"sync/atomic"
)

type atomicFloatCAS2 struct{ u64 uint64 }

func NewAtomicFloatCAS2(initial float64) *atomicFloatCAS2 {
	return &atomicFloatCAS2{u64: math.Float64bits(initial)}
}

// Add attempts to add delta to the value stored in the atomic float and return
// the new value.
func (a *atomicFloatCAS2) Add(delta float64) float64 {
loop:
	oldBits := atomic.LoadUint64(&a.u64)
	newValue := math.Float64frombits(oldBits) + delta
	newBits := math.Float64bits(newValue)
	if !atomic.CompareAndSwapUint64(&a.u64, oldBits, newBits) {
		goto loop
	}
	return newValue
}

// Load atomically loads the current atomic float value.
func (a *atomicFloatCAS2) Load() float64 {
	return math.Float64frombits(atomic.LoadUint64(&a.u64))
}

// Store atomically stores new into the atomic float.
func (a *atomicFloatCAS2) Store(new float64) {
	atomic.StoreUint64(&a.u64, math.Float64bits(new))
}

// Swap atomically stores new and returns the previous value.
func (a *atomicFloatCAS2) Swap(new float64) float64 {
	return math.Float64frombits(atomic.SwapUint64(&a.u64, math.Float64bits(new)))
}
