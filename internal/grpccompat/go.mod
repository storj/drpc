module storj.io/drpc/internal/grpccompat

go 1.18

require (
	github.com/improbable-eng/grpc-web v0.15.0
	github.com/zeebo/assert v1.3.0
	github.com/zeebo/errs v1.2.2
	google.golang.org/grpc v1.50.0
	google.golang.org/protobuf v1.28.1
	storj.io/drpc v0.0.0-00010101000000-000000000000
)

require (
	github.com/cenkalti/backoff/v4 v4.1.1 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/klauspost/compress v1.11.7 // indirect
	github.com/rs/cors v1.8.0 // indirect
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d // indirect
	golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/genproto v0.0.0-20210126160654-44e461bb6506 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

replace storj.io/drpc => ../..
