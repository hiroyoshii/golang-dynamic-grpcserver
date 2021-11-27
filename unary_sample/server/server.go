package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	yaml "gopkg.in/yaml.v2"
)

var (
	port           = flag.Int("port", 50051, "The server port")
	descfilePath   = "unary_sample/helloworld/helloworld_descriptor.pb"
	serverConfPath = "unary_sample/server/server_conf.yaml"
	rpcNameMsgMap  = map[protoreflect.FullName]string{}
)

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadFile(serverConfPath)
	if err != nil {
		panic(err)
	}
	var cnf map[string]map[protoreflect.FullName]string
	err = yaml.Unmarshal(b, &cnf)
	if err != nil {
		panic(err)
	}
	rpcNameMsgMap = cnf["rpc2Msg"]

	s, err := newGrpcServer()
	if err != nil {
		panic(err)
	}
	reflection.Register(s)
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		panic(err)
	}
}

func newGrpcServer() (*grpc.Server, error) {
	s := grpc.NewServer()
	bytes, err := ioutil.ReadFile(descfilePath)
	if err != nil {
		return nil, err
	}
	var fileSet descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(bytes, &fileSet); err != nil {
		return nil, err
	}

	files := protoregistry.GlobalFiles
	for _, fd := range fileSet.File {
		d, err := protodesc.NewFile(fd, files)
		if err != nil {
			return nil, err
		}
		for i := 0; i < d.Services().Len(); i++ {
			s.RegisterService(convertToGrpcDesc(d.Services().Get(i)), nil)
		}
		if err := files.RegisterFile(d); err != nil {
			return nil, err
		}
	}
	return s, nil
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
		if interceptor == nil {
			return callAPI(ctx, in, md.FullName(), md.Output())
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: fmt.Sprintf("/%s/%s", svcd.FullName(), md.Name()),
		}
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return callAPI(ctx, req.(proto.Message), md.FullName(), md.Output())
		}
		return interceptor(ctx, in, info, handler)
	}
}

func callAPI(base context.Context, request proto.Message, fullName protoreflect.FullName, mdesc protoreflect.MessageDescriptor) (interface{}, error) {
	reqbyte, err := protojson.Marshal(request)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(reqbyte, &m)
	if err != nil {
		return nil, err
	}

	msg, ok := rpcNameMsgMap[fullName]
	if !ok {
		return nil, fmt.Errorf("unknown rpc name %s", fullName)
	}
	respbyte := []byte(fmt.Sprintf("{\"message\":\"%s\"}", fmt.Sprintf(msg, m["name"])))
	resp := dynamicpb.NewMessage(mdesc)
	err = protojson.Unmarshal(respbyte, resp)
	return resp, err
}
