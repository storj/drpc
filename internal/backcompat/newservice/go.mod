module storj.io/drpc/internal/backcompat/newservice

go 1.18

require storj.io/drpc/internal/backcompat v0.0.0-00010101000000-000000000000

require (
	github.com/zeebo/errs v1.2.2 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	storj.io/drpc v0.0.0-00010101000000-000000000000 // indirect
	storj.io/drpc/internal/backcompat/servicedefs v0.0.0-00010101000000-000000000000 // indirect
)

replace (
	storj.io/drpc => ../../..
	storj.io/drpc/internal/backcompat => ../
	storj.io/drpc/internal/backcompat/servicedefs => ../newservicedefs
)
