package pincode

import (
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// randomPins returns real pincodes drawn uniformly from the loaded ranges, so CPU
// caches don't flatter the result the way a single repeated pincode would.
func randomPins(n int) []string {
	ensureLoaded()
	r := rand.New(rand.NewSource(1))
	pins := make([]string, 0, n)
	for len(pins) < n {
		if b := buckets[r.Intn(len(buckets))]; b != nil {
			e := b.entries[r.Intn(len(b.entries))]
			pins = append(pins, strconv.Itoa(e.from+r.Intn(e.till-e.from+1)))
		}
	}
	return pins
}

// Typical production case: random valid pincodes.
func BenchmarkLookup(b *testing.B) {
	pins := randomPins(1 << 16)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Lookup(pins[i&(len(pins)-1)])
	}
}

// Best case: the same pincode every time (fully cached).
func BenchmarkLookupSamePincode(b *testing.B) {
	ensureLoaded()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Lookup("560105")
	}
}

// Throughput with all cores looking up concurrently.
func BenchmarkLookupParallel(b *testing.B) {
	pins := randomPins(1 << 16)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Lookup(pins[i&(len(pins)-1)])
			i++
		}
	})
}

// Rejected input (wrong length / letters / leading zero).
func BenchmarkLookupInvalid(b *testing.B) {
	ensureLoaded()
	bad := []string{"12345", "abcdef", "012345", "5601055"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Lookup(bad[i&3])
	}
}

// One-time cost paid by the first Lookup: parsing and indexing the embedded CSV.
// Also reports the heap retained by the loaded index.
func BenchmarkLoad(b *testing.B) {
	ensureLoaded()
	saved := buckets
	defer func() { buckets = saved }()
	var before, after runtime.MemStats
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		buckets = [900]*bucket{}
		runtime.GC()
		runtime.ReadMemStats(&before)
		b.StartTimer()
		if err := load(strings.NewReader(embeddedCSV)); err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		runtime.GC()
		runtime.ReadMemStats(&after)
		b.StartTimer()
	}
	b.ReportMetric(float64(after.HeapAlloc-before.HeapAlloc)/(1<<20), "MB-retained")
}
