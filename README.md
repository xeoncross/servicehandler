
# Service Wrapper

An HTTP mux/router that takes a service and generates the HTTP REST endpoints (with validation) for that service.

The issue with this approach is that services probably shouldn't handle things like access-control or notifications to external services (submitting to a message queue or sending an uptime metric). By having this generate everything, we are losing control over each unique endpoint causing the service we provide to need to handle stuff outside it's domain/scope.


- Ingest service provided
- loop over methods
- for each method
  - create URL endpoint
  - get param types
- on each request, match endpoint (or 404)
- then create param type instances (for absorbing JSON)
- populate with JSON
- validate
- Call method and look for error

## Related

Someone tried a basic version of this before making [gomvc](https://github.com/sdming/gomvc)

# Go doesn't support parameter names

In the following function, there is no way to get the name "page" of the `int` parameter.

    func Foo(page int) {}

This poses a problem for us passing this information along to the client.

- https://groups.google.com/forum/#!topic/golang-nuts/nM_ZhL7fuGc
- https://github.com/golang/go/issues/12296

Possible solutions include:

1. Only using params of a struct type with named fields (which we can read)
2. Creating custom types for every param name (`type page int`)
3. Making sure the params are alphabetical so we can look for the Nth url.Value

### 1. Struct

1. If the request comes in as GET we assume we will find the values in the url.Values
2. If the request comes in as a POST we assume we will find the values in the request.Body

To prevent creating two ways for requesting the same data (GET url params & POST JSON) we will only allow GET requests if the param is an anonymous struct (`struct { a int }`) and vis-versa for POST JSON not allowing GET if the struct is a known type (`type User struct`)

    POST -> u *User
    GET -> params struct{page int}




Also:

- https://stackoverflow.com/questions/31377433/getting-method-parameter-names-in-golang
