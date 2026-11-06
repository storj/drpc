module storj.io/drpc/examples/drpc

go 1.23

require (
	google.golang.org/protobuf v1.36.12
	storj.io/drpc v0.0.17
)

require github.com/zeebo/errs v1.2.2 // indirect

replace storj.io/drpc => ../..
