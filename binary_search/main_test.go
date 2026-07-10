package main

import (
	"testing"
)

// BenchmarkBinarySearch measures the performance of the binary search function.
func BenchmarkBinarySearch(b *testing.B) {
	items := []int{1, 2, 3, 4, 5, 8, 50, 100, 200, 300, 301, 302, 304, 305, 405, 505, 605, 705, 890, 1000}
	target := 505 // Number present in the list

	// ResetTimer ensures the setup time is not included in the benchmark
	b.ResetTimer()

	for b.Loop() { // Loop runs the benchmark b.N times, which is determined by the testing framework for i := 0; i < b.N; i++
		binarySearch(items, target)
	}
}

// TestBinarySearch verifies if the returned index is correct.
func TestBinarySearch(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 8, 50, 100, 200, 300, 301, 302, 304, 305, 405, 505, 605, 705, 890, 1000}
	target := 505
	expected := 15 // The index of the value 505 in the list

	result := binarySearch(items, target)

	if result != expected {
		t.Errorf("Expected index %d, but got %d", expected, result)
	}
}

/*

$ go test -bench=. -benchmem  

Output:

goos: linux
goarch: amd64
pkg: github.com/rodrigochavesoa/int_to_algorithms
cpu: AMD Ryzen 7 5700U with Radeon Graphics         
BenchmarkBinarySearch-16        97912288                12.48 ns/op            0 B/op          0 allocs/op
PASS
ok      github.com/rodrigochavesoa/int_to_algorithms    1.234s

*/