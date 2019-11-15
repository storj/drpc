module storj.io/drpc/internal/grpccompat

go 1.13

require (
	github.com/gogo/protobuf v1.2.1
	github.com/zeebo/assert v1.1.0
	github.com/zeebo/errs v1.2.2
	google.golang.org/grpc v1.25.1
	storj.io/drpc v0.0.4
)

replace storj.io/drpc => ../..
