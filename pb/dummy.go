//go:generate protoc --go-grpc_opt=requireUnimplementedServers=false --go_out=. --go-grpc_out=. echo.proto

package pb
