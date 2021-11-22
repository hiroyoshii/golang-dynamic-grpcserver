package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	port         = flag.Int("port", 50051, "The server port")
	descfilePath = "../helloworld/helloworld_descriptor.pb"
	epNameMsgMap = map[string]string{
		"Greeter.SayHello":   "hello %v",
		"Greeter.SayGoodbye": "good bye %v",
	}
)

// Server is the interface for API Server
type Server interface {
	CallAPI(base context.Context, request proto.Message, endpoint string, md protoreflect.MessageDescriptor) (interface{}, error)
}

// server is used to implement helloworld.GreeterServer.
type server struct {
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer()
	bytes, err := ioutil.ReadFile(descfilePath)
	if err != nil {
		panic(err)
	}
	var fileSet descriptor.FileDescriptorSet
	if err := proto.Unmarshal(bytes, &fileSet); err != nil {
		panic(err)
	}
	// register file descriptor cache for grpcreflection
	gs := &server{}
	files := protoregistry.GlobalFiles
	for _, fd := range fileSet.File {
		d, err := protodesc.NewFile(fd, files)
		if err != nil {
			panic(err)
		}
		for i := 0; i < d.Services().Len(); i++ {
			s.RegisterService(convertToGrpcDesc(d.Services().Get(i)), gs)
		}
		if _, err := files.FindFileByPath(d.Path()); err != nil {
			if err := files.RegisterFile(d); err != nil {
				panic(err)
			}
		}
	}
	reflection.Register(s)
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		panic(err)
	}
}

func convertToGrpcDesc(svcd protoreflect.ServiceDescriptor) *grpc.ServiceDesc {
	gsd := &grpc.ServiceDesc{
		ServiceName: string(svcd.FullName()),
		HandlerType: (map[string]interface{})(nil),
		Streams:     []grpc.StreamDesc{},
		Metadata:    string(svcd.ParentFile().Name()),
	}
	for i := 0; i < svcd.Methods().Len(); i++ {
		md := svcd.Methods().Get(i)
		gsd.Methods = append(
			gsd.Methods, grpc.MethodDesc{
				MethodName: string(md.Name()),
				Handler:    unaryHandler(svcd, md),
			},
		)
	}
	return gsd
}

func unaryHandler(svcd protoreflect.ServiceDescriptor, md protoreflect.MethodDescriptor) func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
		in := dynamicpb.NewMessage(md.Input())
		if err := dec(in); err != nil {
			return nil, err
		}
		epName := fmt.Sprintf("%s.%s", svcd.Name(), md.Name())
		if interceptor == nil {
			return srv.(Server).CallAPI(ctx, in, epName, md.Output())
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: fmt.Sprintf("/%s/%s", svcd.FullName(), md.Name()),
		}
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.(Server).CallAPI(ctx, req.(proto.Message), epName, md.Output())
		}
		return interceptor(ctx, in, info, handler)
	}
}

func (s *server) CallAPI(base context.Context, request proto.Message, endpoint string, mdesc protoreflect.MessageDescriptor) (interface{}, error) {
	reqbyte, err := protojson.Marshal(request)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(reqbyte, &m)
	if err != nil {
		return nil, err
	}

	resp := dynamicpb.NewMessage(mdesc)
	respbyte := []byte(fmt.Sprintf("{\"message\":\"%s\"}", fmt.Sprintf(epNameMsgMap[endpoint], m["name"])))
	err = protojson.Unmarshal(respbyte, resp)
	return resp, err
}
