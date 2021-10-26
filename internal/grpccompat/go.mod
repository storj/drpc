module storj.io/drpc/internal/grpccompat

go 1.17

require (
	github.com/improbable-eng/grpc-web v0.14.0
	github.com/zeebo/assert v1.3.0
	github.com/zeebo/errs v1.2.2
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.26.0
	storj.io/drpc v0.0.0-00010101000000-000000000000
)

require (
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/golang/protobuf v1.5.0 // indirect
	github.com/klauspost/compress v1.10.3 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/rs/cors v1.8.0 // indirect
	golang.org/x/net v0.0.0-20190311183353-d8887717615a // indirect
	golang.org/x/sys v0.0.0-20200116001909-b77594299b42 // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

replace storj.io/drpc => ../..
