module storj.io/drpc/internal/backcompat/newservicedefs

go 1.18

require (
	google.golang.org/protobuf v1.27.1
	storj.io/drpc v0.0.0-00010101000000-000000000000
)

require github.com/zeebo/errs v1.2.2 // indirect

replace storj.io/drpc => ../../..
