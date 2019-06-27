
# Service Wrapper

An HTTP mux/router that takes a service and generates the HTTP REST endpoints (with validation) for that service.

This is great for prototyping projects quickly when writing an external client (i.e. javascript frontend) that will be interacting with your business logic.

**[Example Application](https://github.com/Xeoncross/servicehandler/blob/master/example/README.md)**

This project assumes you follow some sort of domain design such as [Clean Architecture](https://medium.com/@eminetto/clean-architecture-using-golang-b63587aa5e3f). If you still follow the old PHP/Ruby "MVC" approach to websites, this library is not for you.

In Go, you are encouraged to use interfaces so each component does not have to worry about the implementation details of other components. For example, whether users (below) are stored in memory or MySQL should not be a concern for your "service" business logic.

```go
// Our database
memoryStore := NewMemoryStore()

// Our business/domain logic
userService := &UserService{memoryStore}

// Our HTTP handlers (MVC "controllers") are created for us
handler, err := servicehandler.Wrap(userService)

...

log.Fatal(http.ListenAndServe(":8080", handler))
```

This is called [Dependency injection](https://medium.com/@zach_4342/dependency-injection-in-golang-e587c69478a8).

For a complete example, see the [Example Application](https://github.com/Xeoncross/servicehandler/blob/master/example/README.md).

## Usage

Create a service/struct type with functions that look like this:

```go
type UserService struct {
	...
}

func (s *UserService) Create(ctx context.Context, u *User) (int32, error) {
	...
}

func (s *UserService) Find(ctx context.Context, params struct {
	ID int32 `valid:"required"`
}) (*User, error) {
	...
}

func (s *UserService) Foo(ctx context.Context, struct{}) error {}

// etc...

```

### Each method should accept two parameters:

`func(ctx context.Context, params interface{})`

1) `context.Context` from the http.Request
2) a `struct{}` or `&struct{}` pointer with fields describing the validation

For example, in the above service you might be referencing an entity like:

```
type User struct {
	ID    int32  `valid:"int32"`
	Name  string `valid:"alphanum,required"`
	Email string `valid:"email,required"`
}
```

With validation defined using [govalidator](https://godoc.org/github.com/asaskevich/govalidator#ValidateStruct).


### Each method should return one of the following outputs:

```
func (...) error
func (...) (interface{}, error)
```

Whatever interface{} value you return will be JSON encoded and sent to the user
as the response.

## Internal Logic

1. If the request comes in as GET we assume we will find the values in the `url.Values`
2. If the request comes in as a POST we assume we will find the values in the `request.Body`

To prevent creating two ways for requesting the same data (url params & JSON) we will only allow GET requests if the param is an anonymous struct (`struct { a int }`) and vis-versa for POST JSON not allowing GET if the struct is a known type (`type User struct`)

```
POST -> *MyStructType{...}
GET -> struct{...}
```

Please note, struct fields must be public (Capitalized). [govalidator](https://godoc.org/github.com/asaskevich/govalidator#ValidateStruct) will not try to validate private fields. Make sure all struct fields are public.

    a := &struct {
      email string `valid:"email,required"`
    }{}

    ok, err := govalidator.ValidateStruct(a)
    // passes because "email" is not public, so not validated


## Benchmarks

    go test -bench=. --benchmem

This package strongly relies on the [reflect package](https://godoc.org/reflect). However, I still achieve _100k requests per second_ on each CPU core.

```
goos: darwin
goarch: amd64
pkg: github.com/Xeoncross/servicehandler
BenchmarkHandler   	  100000	     11391 ns/op	    4498 B/op	      50 allocs/op
```

The testing I did on [xeoncross/mid](https://github.com/Xeoncross/mid) places this somewhere between [gongular](https://github.com/mustafaakin/gongular) and [echo's Bind()](https://echo.labstack.com/guide/request) (neither of which have the same scope as this project).

## Related

- https://github.com/google/jsonapi
