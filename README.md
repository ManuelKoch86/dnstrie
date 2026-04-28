# trie

A generic, concurrent trie (prefix tree) for Go, designed for **reverse-domain-style key lookups** — e.g. `com.example.api`. Supports exact lookup, deletion, wildcard matching, and is safe for concurrent use via per-node `sync.RWMutex` locking.

---

## Features

- **Generic** — stores any value type via `Trie[T any]`
- **Concurrent** — fine-grained per-node read/write locking
- **Wildcard matching** — `*` segments fan out across all children at that level
- **Automatic pruning** — `Del` cleans up empty ancestor nodes
- **JSON-serializable nodes** — struct fields are exported for marshaling

---

## Installation

```bash
go get github.com/ManuelKoch86/dnstrie
```

> Requires **Go 1.26+** (generics).

---

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/ManuelKoch86/dnstrie"
)

func main() {
    t := trie.New[string]()

    // Set values
    t.Set("com.example.api", "API Service")
    t.Set("com.example.web", "Web Service")
    t.Set("com.other.api",   "Other API")

    // Exact lookup
    val, ok := t.Get("com.example.api")
    fmt.Println(val, ok) // "API Service" true

    // Wildcard match — all nodes under "com.example"
    results := t.Match("*.example.com")
    fmt.Println(results) // ["API Service", "Web Service"]

    // Delete
    t.Del("com.example.api")
    _, ok = t.Get("com.example.api")
    fmt.Println(ok) // false
}
```

---

## Key Design: Reverse-Domain Traversal

Keys are parsed **right to left**, splitting on `.`. The string `"com.example.api"` is stored as:

```
root
 └── "api"
      └── "example"
           └── "com"  ← leaf node, holds value
```

This mirrors DNS-style hierarchies and makes it natural to query sub-trees by domain segment.

---

## API

```go
func New[T any]() Trie[T]
```
Returns a new, empty trie root node.

---

### `Set(uri string, value T)`

Inserts or updates a value at the given key. Creates intermediate nodes as needed.

```go
t.Set("com.example.service", myValue)
```

---

### `Get(uri string) (T, bool)`

Returns the value stored at the exact key, or the zero value and `false` if not found.

```go
val, ok := t.Get("com.example.service")
```

---

### `Del(uri string)`

Removes the value at the given key. After deletion, any ancestor nodes that are no longer needed (no children, no value) are pruned automatically.

```go
t.Del("com.example.service")
```

---

### `Match(uri string) []T`

Returns all values whose keys match the given pattern. A `*` segment acts as a **wildcard**, matching all children at that level and recursing into them.

```go
// Match everything under any sub-domain of "example"
results := t.Match("*.example.com")

// Match all top-level "api" nodes regardless of parent
results = t.Match("api.*")
```

Wildcards are resolved left-to-right in the pattern and expand at each matching level of the tree.

---

## Concurrency

Each node in the tree carries its own `sync.RWMutex`:

- `Get` and `Match` acquire **read locks** (`RLock`) per node traversed.
- `Set` acquires **write locks** (`Lock`) per node to insert or create branches.
- `Del` uses a **mixed strategy**: write lock to unmark the leaf, then a double-checked lock-and-re-check pattern when pruning ancestors to avoid TOCTOU races.

Locks are always released before advancing to the next node, so no more than one node is locked at a time. This means the traversal path is not atomically consistent under concurrent modifications — acceptable for most routing and caching workloads.

---

## Serialization

Node fields (`Branches`, `IsLeaf`, `Key`, `Value`) are exported and tagged for JSON:

```go
type node[T any] struct {
    Branches map[string]*node[T] `json:"b,omitempty"`
    IsLeaf   bool                `json:"l,omitempty"`
    Key      string              `json:"k,omitempty"`
    Value    T                   `json:"v,omitempty"`
    // ...
}
```

> **Note:** `mutex` and `parent` are unexported and will not survive a JSON round-trip. After deserializing, `parent` pointers must be reconstructed before calling `Del`, as it relies on them for ancestor pruning.

---

## Limitations

- **No depth limit on `Match`** — wildcard patterns on very deep trees may recurse significantly. In practice, domain-style keys are shallow.
- **JSON round-trip requires parent reconstruction** — deserialized trees cannot call `Del` safely without re-linking parent pointers.
- **No ordered iteration** — branch maps are unordered; `Match` result order is non-deterministic.

---

## License

MIT — see [LICENSE](./LICENSE).
