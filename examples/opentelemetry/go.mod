module storj.io/drpc/examples/opentelemetry

go 1.26.0

require (
	go.opentelemetry.io/otel v1.47.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.10.0
	go.opentelemetry.io/otel/sdk v1.47.0
	google.golang.org/protobuf v1.36.12
	storj.io/drpc v0.0.17
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/zeebo/errs v1.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace storj.io/drpc => ../..
