module example.com/rpccgo-connect

go 1.24.4

require (
	connectrpc.com/connect v1.19.1
	golang.org/x/net v0.48.0
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
	github.com/ygrpc/rpccgo v0.0.0
)

require (
	github.com/magefile/mage v1.15.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
)

replace github.com/ygrpc/rpccgo => ../..

tool github.com/magefile/mage
