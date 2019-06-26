package main

import "errors"
import "sync/atomic"

// Basic "database" that stores users in memory

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users: make(map[int32]*User),
	}
}

type MemoryStore struct {
	sequence int32
	users    map[int32]*User
}

func (s *MemoryStore) Save(u *User) (int32, error) {
	if u.ID == 0 {
		u.ID = int32(atomic.AddInt32(&s.sequence, 1))
	}

	s.users[u.ID] = u
	return u.ID, nil
}

func (s *MemoryStore) GetByID(id int32) (*User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, errors.New("User not found")
	}
	return user, nil
}
