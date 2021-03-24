module storj.io/drpc/internal/grpccompat

go 1.13

require (
	github.com/zeebo/assert v1.3.0
	github.com/zeebo/errs v1.2.2
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.26.0
	storj.io/drpc v0.0.0-00010101000000-000000000000
)

replace storj.io/drpc => ../..
