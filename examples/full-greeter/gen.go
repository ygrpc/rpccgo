package full

//go:generate protoc -I proto --go_out=proto --go_opt=paths=source_relative --connect-go_out=proto --connect-go_opt=paths=source_relative --connect-go_opt=package_suffix= --connect-go_opt=simple=true --rpc-cgo_out=proto --rpc-cgo_opt=paths=source_relative --rpc-cgo_opt=cgo_dir=../cmd/rpc proto/greeter.proto
