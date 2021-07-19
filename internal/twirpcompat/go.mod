module storj.io/drpc/internal/twirpcompat

go 1.13

require (
	github.com/pkg/errors v0.9.1 // indirect
	github.com/twitchtv/twirp v8.1.0+incompatible
	github.com/zeebo/assert v1.3.0
	github.com/zeebo/hmux v0.3.1
	google.golang.org/protobuf v1.27.1
	storj.io/drpc v0.0.0-00010101000000-000000000000
)

replace storj.io/drpc => ../..
