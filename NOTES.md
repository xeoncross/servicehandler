# Notes

Background and developer notes about the project and codebase.

## Roadmap

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
