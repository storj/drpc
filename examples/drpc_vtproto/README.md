# DRPC example

This example is a bare-bones DRPC use case. It is intended
to show the minimal differences from the gRPC basic example.


```
protoc --go_out=paths=source_relative:. --plugin protoc-gen-go="${GOPATH}/bin/protoc-gen-go" \
  --go-grpc_out=paths=source_relative:. --plugin protoc-gen-go-grpc="${GOPATH}/bin/protoc-gen-go-grpc" \
	--go-drpc_out=paths=source_relative:. --plugin protoc-gen-go-drpc="${GOPATH}/bin/protoc-gen-go-drpc" \
	--go-vtproto_out=paths=source_relative:. --plugin protoc-gen-go-vtproto="${GOPATH}/bin/protoc-gen-go-vtproto" \
	--go-vtproto_opt=pool=storj.io/drpc/examples/drpc_vtproto/pb.Cookie \
	--go-vtproto_opt=pool=storj.io/drpc/examples/drpc_vtproto/pb.CookiePool \
	--go-vtproto_opt=features=marshal+unmarshal+equal+clone+size+pool \
	sesamestreet.proto
```
