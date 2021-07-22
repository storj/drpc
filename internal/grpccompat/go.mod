module storj.io/drpc/internal/grpccompat

go 1.13

require (
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/improbable-eng/grpc-web v0.14.0
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/rs/cors v1.8.0 // indirect
	github.com/zeebo/assert v1.3.0
	github.com/zeebo/errs v1.2.2
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.26.0
	nhooyr.io/websocket v1.8.7 // indirect
	storj.io/drpc v0.0.0-00010101000000-000000000000
)

replace storj.io/drpc => ../..
