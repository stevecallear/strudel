# Strudel
[![Build Status](https://github.com/stevecallear/strudel/actions/workflows/build.yml/badge.svg)](https://github.com/stevecallear/strudel/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/stevecallear/strudel/branch/master/graph/badge.svg)](https://codecov.io/gh/stevecallear/strudel)
[![Go Report Card](https://goreportcard.com/badge/github.com/stevecallear/strudel)](https://goreportcard.com/report/github.com/stevecallear/strudel)

Strudel provides structured error handling and logging middleware for use with [Janice](https://github.com/stevecallear/janice) handlers. It uses the excellent [Logrus](https://github.com/sirupsen/logrus), [jsend](https://github.com/gamegos/jsend) and [httpsnoop](https://github.com/felixge/httpsnoop) packages.

The package is intended to help reduce the initial middleware boilerplate when building an HTTP API. Realistically, error handing and logging requirements will change as an app develops, so the intention here is to simply provide drop in/out components that allow rapid prototyping.

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
	chain := janice.New(strudel.ErrorHandling)

	mux := http.NewServeMux()
	mux.Handle("/", chain.Then(func(w http.ResponseWriter, r *http.Request) error {
		return strudel.NewError("resource not found").
			WithCode(http.StatusNotFound).
			WithField("resourceId", "abc123").
			WithLogField("sensitiveId", "cde456")
	}))

	h := janice.New(strudel.RequestTracking, strudel.Recovery, strudel.RequestLogging).Then(janice.Wrap(mux))
	http.ListenAndServe(":8080", h)
}
```