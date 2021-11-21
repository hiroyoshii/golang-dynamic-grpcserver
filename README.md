# golang-dynamic-grpcserver

extends
https://github.com/grpc/grpc-go/tree/master/examples/helloworld


protoc --include_imports --descriptor_set_out=./unary_sample/helloworld/helloworld_descriptor.pb ./unary_sample/helloworld/hello_world.proto