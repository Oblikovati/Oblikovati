// SPDX-License-Identifier: GPL-2.0-only

package filestore

// MemStore is the shared in-memory fake for tests of filestore-backed stores — the
// named fake the per-package copies duplicated (#1651), e.g.
//
//	store := filestore.NewMemStore[userprefs.Prefs]()
type MemStore[T any] struct {
	value T
	Saved int // number of Save calls, for assertions
}

// NewMemStore returns an empty in-memory store.
func NewMemStore[T any]() *MemStore[T] { return &MemStore[T]{} }

// Load returns the last saved value; found reports whether Save was ever called
// (mirroring FileStore's missing-file semantics).
func (s *MemStore[T]) Load() (T, bool, error) { return s.value, s.Saved > 0, nil }

// Save records v and counts the call.
func (s *MemStore[T]) Save(v T) error {
	s.value = v
	s.Saved++
	return nil
}

// KeyedMemStore is the shared in-memory fake for per-document keyed stores
// (the viewstate shape: one YAML file holding a docKey→T map), e.g.
//
//	store := filestore.NewKeyedMemStore[viewstate.ViewState]()
type KeyedMemStore[T any] struct{ byKey map[string]T }

// NewKeyedMemStore returns an empty keyed in-memory store.
func NewKeyedMemStore[T any]() *KeyedMemStore[T] {
	return &KeyedMemStore[T]{byKey: map[string]T{}}
}

// Load returns the value stored for key, or found=false if there is none.
func (s *KeyedMemStore[T]) Load(key string) (T, bool, error) {
	v, ok := s.byKey[key]
	return v, ok, nil
}

// Save writes (or replaces) key's value.
func (s *KeyedMemStore[T]) Save(key string, v T) error {
	s.byKey[key] = v
	return nil
}
