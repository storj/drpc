module storj.io/drpc/internal/backcompat/oldservice

go 1.16

require storj.io/drpc/internal/backcompat v0.0.0-00010101000000-000000000000

replace (
	storj.io/drpc => storj.io/drpc v0.0.17
	storj.io/drpc/internal/backcompat => ../
	storj.io/drpc/internal/backcompat/servicedefs => ../oldservicedefs
)
