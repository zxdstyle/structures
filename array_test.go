package structures

import (
	"testing"
)

var (
	anyArray = NewArray[int]()
)

func Benchmark_AnyArray_Add(b *testing.B) {
	for i := 0; i < b.N; i++ {
		anyArray.Append(i)
	}
}
