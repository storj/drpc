module storj.io/drpc/internal/backcompat

go 1.18

require (
	github.com/zeebo/assert v1.3.0
	github.com/zeebo/errs v1.2.2
	storj.io/drpc v0.0.0-00010101000000-000000000000
	storj.io/drpc/internal/backcompat/servicedefs v0.0.0-00010101000000-000000000000
)

replace (
	storj.io/drpc => ../..
	storj.io/drpc/internal/backcompat/servicedefs => ./servicedefs
)
