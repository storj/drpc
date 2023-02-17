module storj.io/drpc/examples/opentelemetry

go 1.18

require (
	go.opentelemetry.io/otel v1.10.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.10.0
	go.opentelemetry.io/otel/sdk v1.10.0
	google.golang.org/protobuf v1.27.1
	storj.io/drpc v0.0.17
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/zeebo/errs v1.2.2 // indirect
	go.opentelemetry.io/otel/trace v1.10.0 // indirect
	golang.org/x/sys v0.0.0-20210423185535-09eb48e85fd7 // indirect
)

replace storj.io/drpc => ../..
