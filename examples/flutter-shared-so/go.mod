module example.com/rpccgo-flutter-shared-so

go 1.24.4

require (
	connectrpc.com/connect v1.19.1
	github.com/ygrpc/rpccgo v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/magefile/mage v1.17.2 // indirect
	golang.org/x/sys v0.41.0 // indirect
	google.golang.org/grpc v1.79.3 // indirect
)

replace github.com/ygrpc/rpccgo => ../..

tool github.com/magefile/mage
