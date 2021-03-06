# Go gRPC Middleware

[![Travis Build](https://travis-ci.org/mwitkow/go-grpc-middleware.svg?branch=master)](https://travis-ci.org/mwitkow/go-grpc-middleware)
[![Go Report Card](https://goreportcard.com/badge/github.com/mwitkow/go-grpc-middleware)](https://goreportcard.com/report/github.com/mwitkow/go-grpc-middleware)
[![GoDoc](http://img.shields.io/badge/GoDoc-Reference-blue.svg)](https://godoc.org/github.com/mwitkow/go-grpc-middleware)
[![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

[gRPC Go](https://github.com/grpc/grpc-go) Middleware: interceptors, helpers, utilities.


## Middleware

[gRPC Go](https://github.com/grpc/grpc-go) recently acquired support for
Interceptors, i.e. [middleware](https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81#.gv7tdlghs) 
that is executed either on the gRPC Server before the request is passed onto the user's application logic, or on the gRPC client either around the user call. It is a perfect way to implement
common patters: auth, logging, message, validation, retries or monitoring.

These are generic building blocks that make it easy to build multiple microservices easily.
The purpose of this repository is to act as a go-to point for such reusable functionality. It contains
some of them itself, but also will link to useful external repos.

`grpc_middleware` itself provides support for chaining interceptors. Se [Documentation](doc.md), but here's a simple example:

```go
import "github.com/mwitkow/go-grpc-middleware"

myServer := grpc.NewServer(
    grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(loggingStream, monitoringStream, authStream)),
    grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(loggingUnary, monitoringUnary, authUnary),
)
```

## Interceptors

*Please send a PR to add new interceptors or middleware to this list*

#### Auth
   * [`grpc_auth`](auth/README.md) - a customizable (via `AuthFunc) piece of auth middleware 

#### Logging

#### Monitoring
   * [`grpc_prometheus`⚡](https://github.com/grpc-ecosystem/go-grpc-prometheus) - Prometheus client-side and server-side monitoring middleware
   * [`otgrpc`⚡](https://github.com/grpc-ecosystem/grpc-opentracing/tree/master/go/otgrpc) - [OpenTracing](http://opentracing.io/) client-side and server-side interceptors

#### Client
   * [`grpc_retry`](`retry/README.md`) - a generic gRPC response code retry mechanism

#### Server
   * [`grpc_validator`](validator/README.md) - codegen inbound message validation from `.proto` options 


## Status

This code has been inspired by the gRPC interceptor design discussion, and the [upstream PR](https://github.com/grpc/grpc-go/pull/653).

This code has been running in *production* since May 2016 as the basis of the gRPC micro services stack at [Improbable](https://improbable.io).

Additional tooling will be added, and contributions are welcome.

## License

`go-grpc-middleware` is released under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
