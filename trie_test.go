// Package dnstrie
package dnstrie

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"
)

func GenerateKeys(mask string, count, split uint32) (keys []string) {
	keys = make([]string, count)
	for i := range count {
		keys[i] = fmt.Sprintf(mask, i, i%split)
	}
	return
}

func HasDuplicates[T comparable](slice []T) bool {
	seen := make(map[T]struct{})
	for _, v := range slice {
		if _, ok := seen[v]; ok {
			return true
		}
		seen[v] = struct{}{}
	}
	return false
}

func TestTrie(t *testing.T) {
	const (
		Limit      uint32 = 1000
		Mask       string = "svc-%v.ns-%v.dev.local"
		SearchMask string = "*.ns-%v.dev.local"
		Split      uint32 = 100
	)

	var (
		Keys []string     = GenerateKeys(Mask, Limit, Split)
		Root Trie[uint32] = New[uint32]()
	)

	t.Run("insert", func(t *testing.T) {
		for i := range Limit {
			Root.Set(Keys[i], i)
		}
	})

	t.Run("update", func(t *testing.T) {
		for i := range Limit {
			Root.Set(Keys[i], i+1)
		}
	})

	t.Run("read", func(t *testing.T) {
		var (
			found bool
			val   uint32
		)
		for i := range Limit {
			if val, found = Root.Get(Keys[i]); !found {
				t.Errorf("record not found")
				t.FailNow()
			} else if val != i+1 {
				t.Errorf("record invalid, got: %v expected: %v", val, i+1)
				t.FailNow()
			}
		}
	})

	t.Run("search", func(t *testing.T) {
		var results []uint32
		var searchi uint32 = 15
		results = Root.Match(fmt.Sprintf(SearchMask, searchi))
		t.Logf("i: %v => count: %v mask: %v\r\n", searchi, len(results), fmt.Sprintf(SearchMask, searchi))

		if HasDuplicates(results) {
			t.Errorf("values not unique")
			t.FailNow()
		}

		for i2 := range Limit {
			if i2%Split == searchi {
				if val, found := Root.Get(fmt.Sprintf(Mask, i2, searchi)); !found {
					t.Errorf("key not found")
					t.FailNow()
				} else if !slices.Contains(results, val) {
					t.Errorf("result not found")
					t.FailNow()
				}
			}
		}
	})

	t.Run("delete", func(t *testing.T) {
		for i := range Limit {
			Root.Del(Keys[i])
			if _, found := Root.Get(Keys[i]); found {
				t.Fatalf("key still exists after delete")
			}
		}
	})

	e := json.NewEncoder(t.Output())
	e.SetIndent("", "  ")
	e.Encode(Root)
}

func BenchmarkTrie(b *testing.B) {
	const (
		Mask       string = "svc-%v.ns-%v.dev.local"
		SearchMask string = "*.ns-%v.dev.local"
	)
	Root := New[uint32]()

	b.ResetTimer()
	b.Run("insert", func(b *testing.B) {
		for i := uint32(0); int(i) <= b.N; i++ {
			Root.Set(fmt.Sprintf(Mask, i, i%100), i)
		}
	})

	b.ResetTimer()
	b.Run("update", func(b *testing.B) {
		for i := uint32(0); int(i) <= b.N; i++ {
			Root.Set(fmt.Sprintf(Mask, i, i%100), i+1)
		}
	})

	b.ResetTimer()
	b.Run("read", func(b *testing.B) {
		for i := uint32(0); int(i) <= b.N; i++ {
			Root.Get(fmt.Sprintf(Mask, i, i%100))
		}
	})

	b.ResetTimer()
	b.Run("search", func(b *testing.B) {
		for i := uint32(0); int(i) <= b.N; i++ {
			Root.Match(fmt.Sprintf(SearchMask, i%100))
		}
	})

	b.ResetTimer()
	b.Run("delete", func(b *testing.B) {
		for i := uint32(0); int(i) <= b.N; i++ {
			Root.Del(fmt.Sprintf(Mask, i, i%100))
		}
	})
}
