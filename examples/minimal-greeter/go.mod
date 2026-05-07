module example.com/rpccgo-minimal

go 1.24.4

require (
	connectrpc.com/connect v1.19.1
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
	rpccgo v0.0.0
)

require github.com/magefile/mage v1.15.0 // indirect

replace rpccgo => ../..
