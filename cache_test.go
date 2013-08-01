package cache

import (
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

type TestStruct struct {
	Num      int
	Children []*TestStruct
}

func TestCache(t *testing.T) {
	tc := New(0)

	a, found := tc.Get("a")
	if found || a != "" {
		t.Error("Getting A found value that shouldn't exist:", a)
	}

	b, found := tc.Get("b")
	if found || b != "" {
		t.Error("Getting B found value that shouldn't exist:", b)
	}

	c, found := tc.Get("c")
	if found || c != "" {
		t.Error("Getting C found value that shouldn't exist:", c)
	}

	tc.Set("a", "1", 0)
	tc.Set("b", "b", 0)
	tc.Set("c", "3.5", 0)

	x, found := tc.Get("a")
	if !found {
		t.Error("a was not found while getting a2")
	}
	if x == "" {
		t.Error("x for a is nil")
	} else if a2, _ := strconv.ParseInt(x, 10, 0); a2+2 != 3 {
		t.Error("a2 (which should be 1) plus 2 does not equal 3; value:", a2)
	}

	x, found = tc.Get("b")
	if !found {
		t.Error("b was not found while getting b2")
	}
	if x == "" {
		t.Error("x for b is nil")
	} else if x+"B" != "bB" {
		t.Error("b2 (which should be b) plus B does not equal bB; value:", x)
	}

	x, found = tc.Get("c")
	if !found {
		t.Error("c was not found while getting c2")
	}
	if x == "" {
		t.Error("x for c is nil")
	} else if c2, _ := strconv.ParseFloat(x, 32); c2+1.2 != 4.7 {
		t.Error("c2 (which should be 3.5) plus 1.2 does not equal 4.7; value:", c2)
	}
}

func TestAdd(t *testing.T) {
	tc := New(0)
	err := tc.Add("foo", "bar", 0)
	if err != nil {
		t.Error("Couldn't add foo even though it shouldn't exist")
	}
	err = tc.Add("foo", "baz", 0)
	if err == nil {
		t.Error("Successfully added another foo when it should have returned an error")
	}
}

func TestReplace(t *testing.T) {
	tc := New(0)
	err := tc.Replace("foo", "bar", 0)
	if err == nil {
		t.Error("Replaced foo when it shouldn't exist")
	}
	tc.Set("foo", "bar", 0)
	time.Sleep(100 * time.Millisecond)
	err = tc.Replace("foo", "bar", 0)
	if err != nil {
		t.Error("Couldn't replace existing key foo")
	}
}

func TestDelete(t *testing.T) {
	tc := New(0)
	tc.Set("foo", "bar", 0)
	tc.Delete("foo")
	time.Sleep(100 * time.Millisecond)
	x, found := tc.Get("foo")
	if found {
		t.Error("foo was found, but it should have been deleted")
	}
	if x != "" {
		t.Error("x is not nil:", x)
	}
}

func TestItemCount(t *testing.T) {
	tc := New(0)
	tc.Set("foo", "1", 0)
	tc.Set("bar", "2", 0)
	tc.Set("baz", "3", 0)
	if n := tc.ItemCount(); n != 3 {
		t.Errorf("Item count is not 3: %d", n)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	b.StopTimer()
	tc := New(0)
	tc.Set("foo", "bar", 0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Get("foo")
	}
}

func BenchmarkRWMutexMapGet(b *testing.B) {
	b.StopTimer()
	m := map[string]string{
		"foo": "bar",
	}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_, _ = m["foo"]
		mu.RUnlock()
	}
}

func BenchmarkCacheGetConcurrent(b *testing.B) {
	b.StopTimer()
	tc := New(0)
	tc.Set("foo", "bar", 0)
	wg := new(sync.WaitGroup)
	workers := runtime.NumCPU()
	each := b.N / workers
	wg.Add(workers)
	b.StartTimer()
	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < each; j++ {
				tc.Get("foo")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkRWMutexMapGetConcurrent(b *testing.B) {
	b.StopTimer()
	m := map[string]string{
		"foo": "bar",
	}
	mu := sync.RWMutex{}
	wg := new(sync.WaitGroup)
	workers := runtime.NumCPU()
	each := b.N / workers
	wg.Add(workers)
	b.StartTimer()
	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < each; j++ {
				mu.RLock()
				_, _ = m["foo"]
				mu.RUnlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkCacheGetManyConcurrent(b *testing.B) {
	// This is the same as BenchmarkCacheGetConcurrent, but its result
	// can be compared against BenchmarkShardedCacheGetManyConcurrent.
	b.StopTimer()
	n := 10000
	tc := New(0)
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		k := "foo" + strconv.Itoa(n)
		keys[i] = k
		tc.Set(k, "bar", 0)
	}
	each := b.N / n
	wg := new(sync.WaitGroup)
	wg.Add(n)
	for _, v := range keys {
		go func() {
			for j := 0; j < each; j++ {
				tc.Get(v)
			}
			wg.Done()
		}()
	}
	b.StartTimer()
	wg.Wait()
}

func BenchmarkCacheSet(b *testing.B) {
	b.StopTimer()
	tc := New(0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", "bar", 0)
	}
}

func BenchmarkRWMutexMapSet(b *testing.B) {
	b.StopTimer()
	m := map[string]string{}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		mu.Unlock()
	}
}

func BenchmarkCacheSetDelete(b *testing.B) {
	b.StopTimer()
	tc := New(0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", "bar", 0)
		tc.Delete("foo")
	}
}

func BenchmarkRWMutexMapSetDelete(b *testing.B) {
	b.StopTimer()
	m := map[string]string{}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		mu.Unlock()
		mu.Lock()
		delete(m, "foo")
		mu.Unlock()
	}
}

func BenchmarkCacheSetDeleteSingleLock(b *testing.B) {
	b.StopTimer()
	tc := New(0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Lock()
		tc.set("foo", "bar", 0)
		tc.delete("foo")
		tc.Unlock()
	}
}

func BenchmarkRWMutexMapSetDeleteSingleLock(b *testing.B) {
	b.StopTimer()
	m := map[string]string{}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		delete(m, "foo")
		mu.Unlock()
	}
}
