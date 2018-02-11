# Strudel
Strudel provides structured error handling and logging middleware for use with [Janice](https://github.com/stevecallear/janice) handlers. It uses the excellent [Logrus](https://github.com/sirupsen/logrus), [jsend](https://github.com/gamegos/jsend) and [httpsnoop](https://github.com/felixge/httpsnoop) packages.

The package is intended to help reduce the initial middleware boilerplate when building an HTTP API. Realistically error handing and logging requirements will change as an app develops, so the intention here is more to provide drop in/out components that allow rapid prototyping.

## Getting started
```
go get github.com/stevecallear/strudel
```

## Example
```
package main

import (
	"net/http"

	"github.com/stevecallear/janice"
	"github.com/stevecallear/strudel"
)

func main() {
	c := janice.New(strudel.ErrorHandling)
	m := http.NewServeMux()
	m.Handle("/", c.Then(func(w http.ResponseWriter, r *http.Request) error {
		return strudel.NewError("resource not found").
			WithCode(http.StatusNotFound).
			WithField("resourceId", "abc123")
	}))
	h := janice.New(strudel.Recovery, strudel.RequestLogging).Then(janice.Wrap(m))
	http.ListenAndServe(":8080", h)
}
```