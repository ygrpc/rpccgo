package minimal

//go:generate protoc -I . --go_out=. --go_opt=module=example.com/rpccgo-minimal --connect-go_out=. --connect-go_opt=module=example.com/rpccgo-minimal --connect-go_opt=package_suffix= --connect-go_opt=simple=true --rpc-cgo_out=. --rpc-cgo_opt=module=example.com/rpccgo-minimal --rpc-cgo_opt=cgo_dir=../../../cmd/rpc proto/greeter.proto
