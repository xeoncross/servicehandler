package main

import "context"

// We are using "Clean Architecture" which means we are not worrying about
// the implementation. Instead focusing on a testable, database independent,
// layer of entities, services, and stores/repositories.
//
// This means we can focus on our app logic - not the server. Please note that
// we have everything served under the same "main" package. In reality you
// would be using subpackages and better organization.
//
// https://medium.com/@eminetto/clean-architecture-using-golang-b63587aa5e3f

// User is our domain model of what a "user" is
type User struct {
	ID    int32  `valid:"int32"`
	Name  string `valid:"alphanum,required"`
	Email string `valid:"email,required"`
}

// UserStore could be a BoltDB, MySQL, Redis, SQLite, or DynamoDB backend
// We don't care as long as it implements this interface
type UserStore interface {
	Save(*User) (int32, error)
	GetByID(int32) (*User, error)
}

// UserService contains actual business logic (regardless of what store is used)
type UserService struct {
	Store UserStore
}

func (s *UserService) Create(ctx context.Context, u *User) (int32, error) {
	// Send welcome email?
	// Log metrics?
	// foobar?
	return s.Store.Save(u)
}

func (s *UserService) Get(ctx context.Context, params struct {
	ID int32 `valid:"required"`
}) (*User, error) {
	user, err := s.Store.GetByID(params.ID)
	if err != nil {
		return nil, err
	}

	// Extra checks here?
	// Ensure we have permissions to access this user?

	return user, nil
}
