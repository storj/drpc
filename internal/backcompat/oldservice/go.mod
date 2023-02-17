module storj.io/drpc/internal/backcompat/oldservice

go 1.18

require storj.io/drpc/internal/backcompat v0.0.0-00010101000000-000000000000

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/spacemonkeygo/monkit/v3 v3.0.7 // indirect
	github.com/zeebo/errs v1.2.2 // indirect
	storj.io/drpc v0.0.17 // indirect
	storj.io/drpc/internal/backcompat/servicedefs v0.0.0-00010101000000-000000000000 // indirect
)

replace (
	storj.io/drpc => storj.io/drpc v0.0.17
	storj.io/drpc/internal/backcompat => ../
	storj.io/drpc/internal/backcompat/servicedefs => ../oldservicedefs
)
