module storj.io/drpc/internal/backcompat/newservice

go 1.16

require storj.io/drpc/internal/backcompat v0.0.0-00010101000000-000000000000

replace (
	storj.io/drpc => ../../..
	storj.io/drpc/internal/backcompat => ../
	storj.io/drpc/internal/backcompat/servicedefs => ../newservicedefs
)
