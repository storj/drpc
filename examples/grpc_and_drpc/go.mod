module storj.io/drpc/examples/grpc_and_drpc

go 1.26.0

require (
	golang.org/x/sync v0.24.0
	google.golang.org/grpc v1.85.0
	google.golang.org/protobuf v1.36.12
	storj.io/drpc v0.0.17
)

require (
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/zeebo/errs v1.2.2 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
)

replace storj.io/drpc => ../..
