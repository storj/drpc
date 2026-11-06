module storj.io/drpc/internal/grpccompat

go 1.26.0

require (
	github.com/improbable-eng/grpc-web v0.15.0
	github.com/zeebo/assert v1.3.0
	github.com/zeebo/errs v1.2.2
	google.golang.org/grpc v1.85.0
	google.golang.org/protobuf v1.36.12
	storj.io/drpc v0.0.0-00010101000000-000000000000
)

require (
	github.com/cenkalti/backoff/v4 v4.1.1 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/klauspost/compress v1.20.3 // indirect
	github.com/rs/cors v1.8.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto v0.0.0-20210126160654-44e461bb6506 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

replace storj.io/drpc => ../..
