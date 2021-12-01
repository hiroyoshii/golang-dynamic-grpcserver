# golang-generic-grpcserver
## overview
samples for generic gRPC server in golang.
Without generated golang server/client from protobuf file, implements gRPC server/client.
## repository tree
```
.
└── unary_sample
    ├── client - generic client of helloworld gRPC
    ├── helloworld - helloworld protobuf file
    └── server - generic server of helloworld gRPC
```

## protobuf file
Based on https://github.com/grpc/grpc-go/tree/master/examples/helloworld, this repository adds SayGoodbye RPC.

## generate descriptor
generate protobuf descriptor_set using following command

```
protoc --include_imports --descriptor_set_out=./unary_sample/helloworld/helloworld_descriptor.pb ./unary_sample/helloworld/hello_world.proto
```

## run sample gRPC server and client
### run server
```
go run unary_sample/server/server.go
```
### run client
```
go run unary_sample/client/client.go --name <name>
```
