# atomic-float experiment

I was a tad bit concerned yesterday when working on a program because
it uses a pseudo atomic float structure to provide global access to a
float64 variable on a heavily used server. Each request checks this
value, so it cannot bog down the service when under higher query
loads.

There are two ways off the top of my head to make a float64 atomic in
Go. The first is with a simple mutex. I'll show this code which should
seem rather benign if you've ever used mutexes in Go before.

```Go
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
```

The second method I can think of to use a pseudo atomic floating point
variable is by using a bit of compare-and-swap (CAS) code.

```Go
type atomicFloatCAS struct {
    u64 uint64
}
```

I'm sure this struct could be eliminated, but what causes most
curiosity is the fact that I'm storing a float64 in a uint64
field. CAS values are fairly useful constructs, but they work with
integers rather than floating point values. Therefore we need to
perform a bit of conversion on the binary representation of the values
before storing and after loading.

```Go
func NewAtomicFloat(initial float64) *atomicFloatCAS {
	return &atomicFloatCAS{u64: math.Float64bits(initial)}
}

// Load atomically loads the current atomic float value.
func (a *atomicFloatCAS) Load() float64 {
	return math.Float64frombits(atomic.LoadUint64(&a.u64))
}

// Store atomically stores new into the atomic float.
func (a *atomicFloatCAS) Store(new float64) {
	atomic.StoreUint64(&a.u64, math.Float64bits(new))
}
```

What is noticably missing in the above set of structure methods for
the CAS version of the structure is the ability to atomically add a
value to the variable, like an accumulator.

```Go
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
```

To understand what's going on, let's ignore the `for` loop for a
moment. You can see the method loads the uint64 bit representation,
oldBits, converts that into a floating point value and adds it to the
delta, newValue, and calculates what the uint64 bit representation
should be for the new value, newBits. Then the function invokes the
CAS instruction. This function uses CPU instructions to guarantee the
entire operation is atomic, and a different thread on a possibly
different CPU will not see any state in while this process executes.

This method asks the CAS function to check the current value of the
memory pointed to by the field. If the value matches oldBits, what we
read from that value a few moments ago, then CAS will overwrite the
memory with newBits. In other words, if nothing else came along and
changed that memory value while newValue or newBits were being
calculated, then we can safely update the memory with newBits,
effectively adding delta to the atomic float. In this case, CAS will
return true, and the function will return the newValue.

However, when CAS returns false, it indicates that something else came
along and changed the 8 bytes in that memory location while newValue
or newBits were being calculated. When this happens, if we were to
wrote newBits on top of the value, we would loose any changes to our
accumulator that other threads may have done. Which would not make for
a very good atomic float. In this case, when CAS returns false, we
know the 8 bytes of memory are no longer oldBits, and we should loop
back up to the top, then re-read what oldBits are, calculate newBits,
and try again.

All of this works quite well, but the concern is the unbounded
looping. If there is only a single thread of execution which ever
attempts to update this atomic float, then every time it invokes the
CAS function, nothing else could have changed the 8 bytes, and CAS
will always perform the atomic swap and return true. But if two
threads of execution were simultaneously trying to update these 8
bytes, each of the threads would get a false half of the times it
invoked CAS.

In fact, the more threads are trying to mutate this value, there is
more contention for updating these 8 bytes. When 10 threads are all
trying to update these 8 bytes, then depending on timing of the
scheduler, up to 9 of them could get a false and have to try again at
any moment. CAS functions do not necessarily scale perfectly well with
high contention.

Which brings me back to my server problem, which needs to have a
floating point value that is atomically updated, by potentially
hundreds or thousands of threads of execution. This begs the question
whether this CAS implementation will scale under high contention as
well as a simple mutex implementation will?

But how could mutex values be faster than CAS? Many people who know
about CAS also know that most schedulers implement mutex locks using
CAS variables. So if a mutex is implemented using a CAS, how could it
be faster than CAS? Well, that is because a proper mutex
implementation, like what we have with the Go runtime, is integrated
with Go's scheduler. The Go scheduler tracks information about which
go routines are runnable and which are not. When a go routine blocks
on a mutex, the scheduler does not schedule any runtime for it. So a
mutex uses practically no resources until the mutex is unlocked.

The Go scheduler has no insight into programs using CAS logic,
however. If a CAS is blocked due to high contention, the Go scheduler
happily continues scheduling that go routine because it's not
blocked. That go routine atomically loads the value, does some
computations on the value, then attempts to store the new value. There
could easily be tens of go routines attempting to do that at a time on
different CPU cores, all but one of them failing, and each of them
getting scheduled to run by the scheduler.

The point is that CAS can be quicker, but under very high contention,
it can sometimes lead to congestion as the Go scheduler schedules go
routines that vie with each other for the opportunity to modify the
CAS value. This can cause a situation where performance bottoms
out. In comparision, while mutex locks have an overall lower
performance, the Go runtime gives priority boosts for go routines that
are locked on a mutex for a long time, so they might get preferential
treatment and also get to make progress.

At any rate, I wanted to test the two methods of a pseudo atomic
floating point variable, and came up with this set of files to do
so. These benchmarks seem relevant to the project I am currently
working on, and might not provide the same result for other projects.
