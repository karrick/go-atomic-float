package atomic

import "sync"

type atomicFloatMutex struct {
	f64 float64
	l   sync.RWMutex
}

func NewAtomicFloatMutex(initial float64) *atomicFloatMutex {
	return &atomicFloatMutex{f64: initial}
}

// Add attempts to add delta to the value stored in the atomic float and return
// the new value.
func (a *atomicFloatMutex) Add(delta float64) float64 {
	a.l.Lock()
	a.f64 += delta
	new := a.f64
	a.l.Unlock()
	return new
}

// Load atomically loads the current atomic float value.
func (a *atomicFloatMutex) Load() float64 {
	a.l.RLock()
	f := a.f64
	a.l.RUnlock()
	return f
}

// Store atomically stores new into the atomic float.
func (a *atomicFloatMutex) Store(new float64) {
	a.l.Lock()
	a.f64 = new
	a.l.Unlock()
}

// Swap atomically stores new and returns the previous value.
func (a *atomicFloatMutex) Swap(new float64) float64 {
	a.l.Lock()
	old := a.f64
	a.f64 = new
	a.l.Unlock()
	return old
}
