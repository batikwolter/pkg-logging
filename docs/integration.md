# Integration Guide (Other Services)

Use this guide to adopt `github.com/deccara-tech/pkg-logging` in another Go service.

## 1. Add dependency

```bash
go get github.com/deccara-tech/pkg-logging@latest
```

## 2. Bootstrap logger at service startup

```go
package main

import (
	"log"

	"github.com/deccara-tech/pkg-logging/observe"
)

func mustLogger() observe.Logger {
	cfg := observe.Config{
		Resource: observe.Resource{
			ServiceName:    "order-service",
			ServiceVersion: "1.0.0",
			Environment:    "prod",
		},
		Level:           "info",
		Format:          "json",
		Output:          "stdout",
		GenerateTraceID: observe.BoolPtr(true), // omit for default=true
	}

	logger, err := observe.New(cfg)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	return logger
}
```

Notes:
- `Resource.ServiceName` and `Resource.Environment` required.
- If `GenerateTraceID` false: package will not synthesize missing `trace.id`.

## 3. Wire gRPC server interceptor

```go
import (
	gogrpc "google.golang.org/grpc"

	"github.com/deccara-tech/pkg-logging/observe"
	observegrpc "github.com/deccara-tech/pkg-logging/observe/grpc"
)

func newServer(logger observe.Logger) *gogrpc.Server {
	return gogrpc.NewServer(
		gogrpc.ChainUnaryInterceptor(
			observegrpc.UnaryServerInterceptor(logger),
		),
	)
}
```

Server interceptor emits one completion log per unary RPC:
- `event=rpc.server.completed`
- `grpc.service`, `grpc.method`, `grpc.status_code`, `duration_ms`
- `request.id`, `correlation.id`, `trace.id` (policy-dependent)

## 4. Wire gRPC client interceptor

```go
import (
	gogrpc "google.golang.org/grpc"

	"github.com/deccara-tech/pkg-logging/observe"
	observegrpc "github.com/deccara-tech/pkg-logging/observe/grpc"
)

func dial(logger observe.Logger, target string) (*gogrpc.ClientConn, error) {
	return gogrpc.Dial(
		target,
		gogrpc.WithInsecure(),
		gogrpc.WithUnaryInterceptor(
			observegrpc.UnaryClientInterceptor(
				logger,
				observegrpc.WithSourceService("order-service"),
			),
		),
	)
}
```

Client interceptor propagates:
- `x-request-id`
- `x-correlation-id`
- `x-trace-id` (policy-dependent)
- `x-source-service`

And emits one completion log per outbound unary RPC:
- `event=rpc.client.completed`

## 5. Use logger in handlers (domain logs)

```go
func (s *Server) ApproveOrder(ctx context.Context, req *pb.ApproveOrderRequest) (*pb.ApproveOrderResponse, error) {
	log := observe.FromContext(ctx)

	log.InfoContext(ctx, "order approved",
		observe.Event("order.approved"),
		observe.String("order.id", req.OrderId),
	)

	return &pb.ApproveOrderResponse{}, nil
}
```

Rule:
- Interceptor owns request completion log.
- Handler owns domain/business logs.

## 6. Redaction behavior

Keys like `authorization`, `cookie`, `password`, `secret`, `token`, `api key` are masked automatically.

Explicit sensitive field:

```go
log.InfoContext(ctx, "auth attempt",
	observe.SensitiveString("auth.password", "raw-secret"),
)
```

## 7. Test integration in service repo

Use test sink to assert schema and propagation:

```go
sink := testkit.NewSink()
logger, _ := observe.New(observe.Config{
	Resource: observe.Resource{ServiceName: "svc", Environment: "test"},
	Writer:   sink,
})

logger.InfoContext(context.Background(), "hello", observe.Event("svc.started"))
entries := sink.Entries()
_ = entries
```

## 8. Rollout checklist

- Replace ad-hoc logger init with `observe.New`.
- Attach server interceptor on all gRPC servers.
- Attach client interceptor on all gRPC outbound clients.
- Ensure handlers use `observe.FromContext(ctx)`.
- Standardize event names (`dot.separated.lowercase`).
- Avoid request/response payload logging by default.
- Add at least one integration test checking IDs + completion log schema.

## Outcome After Integration

You get:
- Consistent org-wide JSON schema across services.
- Automatic request lifecycle logs (server/client) with stable fields.
- Request/correlation/trace continuity across gRPC hops.
- Less duplicate logging between infra and handler code.
- Safer logs via default redaction pass.
- Lower onboarding friction: one shared API, one behavior model.
