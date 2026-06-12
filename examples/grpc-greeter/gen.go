package grpcgreeter

//go:generate protoc --unsafe_allow_out_dir_escape -I . --go_out=. --go_opt=module=example.com/rpccgo-grpc --go-grpc_out=. --go-grpc_opt=module=example.com/rpccgo-grpc --rpc-cgo_out=. --rpc-cgo_opt=module=example.com/rpccgo-grpc --rpc-cgo_opt=cgo_dir=../../../cmd/rpc proto/greeter.proto
