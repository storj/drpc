module storj.io/drpc/examples/drpc_and_http

go 1.18

require (
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/protobuf v1.27.1
	storj.io/drpc v0.0.17
)

require github.com/zeebo/errs v1.2.2 // indirect

replace storj.io/drpc => ../..
