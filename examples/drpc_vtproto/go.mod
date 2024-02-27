module storj.io/drpc/examples/drpc_vtproto

go 1.19

require (
	github.com/planetscale/vtprotobuf v0.6.0
	google.golang.org/grpc v1.62.0
	google.golang.org/protobuf v1.32.0
	storj.io/drpc v0.0.17
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/zeebo/errs v1.2.2 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
)

replace storj.io/drpc => ../..
