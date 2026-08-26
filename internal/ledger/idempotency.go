package ledger

import (
	"encoding/json"
	"sync"
)

func idempotencyMapKey(scope, key string) string {
	return scope + "\x00" + key
}

func cloneIdempotency(value IdempotencyRecord) IdempotencyRecord {
	value.Response = append(json.RawMessage(nil), value.Response...)
	return value
}

func (s *Store) LookupIdempotency(scope, key string) (IdempotencyRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.projection.Idempotency[idempotencyMapKey(scope, key)]
	if !ok {
		return IdempotencyRecord{}, false
	}
	return cloneIdempotency(value), true
}

type KeyLocker struct {
	mu    sync.Mutex
	locks map[string]*keyLock
}

type keyLock struct {
	mu   sync.Mutex
	refs int
}

func NewKeyLocker() *KeyLocker {
	return &KeyLocker{locks: make(map[string]*keyLock)}
}

func (k *KeyLocker) Lock(key string) func() {
	k.mu.Lock()
	entry := k.locks[key]
	if entry == nil {
		entry = &keyLock{}
		k.locks[key] = entry
	}
	entry.refs++
	k.mu.Unlock()
	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		k.mu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}
