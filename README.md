
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

# Internal Logic

1. If the request comes in as GET we assume we will find the values in the `url.Values`
2. If the request comes in as a POST we assume we will find the values in the `request.Body`

To prevent creating two ways for requesting the same data (url params & JSON) we will only allow GET requests if the param is an anonymous struct (`struct { a int }`) and vis-versa for POST JSON not allowing GET if the struct is a known type (`type User struct`)

```
POST -> u *User
GET -> params struct{page int}
```

Please note, struct fields must be public (Capitalized). [govalidator](https://godoc.org/github.com/asaskevich/govalidator#ValidateStruct) will not try to validate private fields. Make sure all struct fields are public.

    a := &struct {
      email string `valid:"email,required"`
    }{}

    ok, err := govalidator.ValidateStruct(a)
    // passes because "email" is not public, so not validated


## Output

Service methods must return results matching one of two ways:

    func (...) error
    func (...) (interface{}, error)

The interface{} is often a struct or []slice of structs and is sent JSON encoded to the client.


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

# Roadmap

Move towards code-generation ([go/parser](https://golang.org/pkg/go/parser/)). This would solve the two biggest issues with this project: 1) custom handler changes and 2) reflect cannot read function parameters/output variable names resulting in the need for wrapping structs.

_Help needed_

## Issue 1: Customization

The issue with using reflect is that services probably shouldn't handle things like access-control or notifications to external services (submitting to a message queue or sending an uptime metric). By having this generate everything, we are losing control over each unique endpoint causing the service we provide to need to handle stuff outside it's domain/scope.

We can still wrap middleware around/before the http.Handler created by `servicehandler.Wrap()`. However we have no control over the `validation -> execute` stage.

## Issue 2: `reflect` does not support [variable names](https://stackoverflow.com/questions/31377433/getting-method-parameter-names-in-golang)

In the following function, there is no way to get the name "page" of the `int` parameter.

    func Foo(page int) {}

This poses a problem for us passing this information along to the client.

- https://groups.google.com/forum/#!topic/golang-nuts/nM_ZhL7fuGc
- https://github.com/golang/go/issues/12296

Possible solutions include:

1. Only using params of a struct type with named fields (which we can read)
2. Creating custom types for every param name (`type page int`)
3. Making sure the params are alphabetical so we can look for the Nth url.Value

Only the first option, wrapping structs, makes any sense.

## Related

- https://github.com/google/jsonapi
