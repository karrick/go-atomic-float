package atomic

import (
	"strconv"
	"sync"
	"testing"
)

type af64 interface {
	Add(float64) float64
	Load() float64
}

func runQ(tb testing.TB, af af64, adderCount, loaderCount, operationCount int) {
	tb.Helper()
	var adderGroup, loaderGroup sync.WaitGroup
	adderGroup.Add(adderCount)
	loaderGroup.Add(loaderCount)

	// spawn adder threads
	for i := 0; i < adderCount; i++ {
		go func() {
			for i := 0; i < operationCount; i++ {
				af.Add(1)
			}
			adderGroup.Done()
		}()
	}

	// spawn loader threads
	for i := 0; i < loaderCount; i++ {
		go func() {
			var sum float64
			for i := 0; i < operationCount; i++ {
				sum += af.Load()
			}
			loaderGroup.Done()
			_ = sum
		}()
	}

	loaderGroup.Wait() // wait for loaders to complete
	adderGroup.Wait()  // wait for the adders to complete

	if got, want := af.Load(), 1*float64(adderCount*operationCount); got != want {
		tb.Errorf("GOT: %v; WANT: %v", got, want)
	}
}

func BenchmarkProducerConsumer(b *testing.B) {
	const itemsPerLoader = 1000

	c := func(b *testing.B, count int) {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			b.Run("cas", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					af := NewAtomicFloatCAS(0)
					runQ(b, af, count, count, itemsPerLoader)
				}
			})
			b.Run("cas2", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					af := NewAtomicFloatCAS2(0)
					runQ(b, af, count, count, itemsPerLoader)
				}
			})
			b.Run("lock", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					af := NewAtomicFloatMutex(0)
					runQ(b, af, count, count, itemsPerLoader)
				}
			})
		})
	}

	c(b, 10)
	c(b, 100)
	c(b, 1000)
	c(b, 10000)
	c(b, 100000)
}
