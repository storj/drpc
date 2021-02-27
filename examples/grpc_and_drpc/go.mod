module storj.io/drpc/examples/grpc_and_drpc

go 1.16

require (
	github.com/golang/protobuf v1.4.3
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.25.0
	storj.io/drpc v0.0.17
)

replace storj.io/drpc => ../..
