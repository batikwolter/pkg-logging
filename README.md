# pkg-logging

Production-ready shared logging foundation for Go services.

Integration doc for adopters: `docs/integration.md`.

## Features

- Structured JSON logging via `observe.Logger` (org-owned interface)
- Stable resource identity fields (`service.name`, `service.version`, `env`)
- Request metadata propagation through `context.Context`
- Flat dotted schema helpers (`Field` constructors)
- Standard operational event constants
- Error classification helpers with gRPC status mapping
- Denylist redaction safety pass
- gRPC unary server/client interceptors
- `observe/testkit` in-memory JSON sink and assertions

## Quick Start

```go
cfg := observe.Config{
    Resource: observe.Resource{
        ServiceName:    "order-service",
        ServiceVersion: "1.0.0",
        Environment:    "prod",
    },
    Level:           "info",
    Format:          "json",
    Output:          "stdout",
    GenerateTraceID: observe.BoolPtr(true), // optional; default true when omitted
}

logger, err := observe.New(cfg)
if err != nil {
    panic(err)
}

server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        observegrpc.UnaryServerInterceptor(logger),
    ),
)
_ = server
```

Handler usage:

```go
log := observe.FromContext(ctx)
log.InfoContext(ctx, "order approved",
    observe.String("event", "order.approved"),
    observe.String("order.id", orderID),
)
```

## Logging Rules

- Interceptors emit one completion log per unary RPC (`rpc.server.completed`, `rpc.client.completed`)
- Handlers emit domain/business events
- No request/response body logging by default
- Expected business failures map to warning-level completion logs
- Unexpected/system failures map to error-level completion logs
- `trace.id` is generated when missing by default; disable with `GenerateTraceID: observe.BoolPtr(false)`

## Tests

```bash
go test ./...
```
