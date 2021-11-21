package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	defaultName = "world"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cli := rpb.NewServerReflectionClient(conn)
	info, err := cli.ServerReflectionInfo(ctx)
	err = info.SendMsg(&rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_FileByFilename{
			FileByFilename: "hello_world.proto",
		},
	})
	resp := &rpb.ServerReflectionResponse{}
	err = info.RecvMsg(resp)
	if err != nil {
		panic(err)
	}
	filepb := &descriptorpb.FileDescriptorProto{}
	for _, fd := range resp.GetFileDescriptorResponse().FileDescriptorProto {
		proto.Unmarshal(fd, filepb)
		filedesc, _ := protodesc.NewFile(filepb, nil)
		svcdesc := filedesc.Services().Get(0)
		for i := 0; i < svcdesc.Methods().Len(); i++ {
			method := svcdesc.Methods().Get(i)
			if !method.IsStreamingServer() {
				req := dynamicpb.NewMessage(method.Input())
				reqbyte := []byte(fmt.Sprintf("{\"name\":\"%s\"}", *name))
				err = protojson.Unmarshal(reqbyte, req)
				if err != nil {
					panic(err)
				}
				res := dynamicpb.NewMessage(method.Output())
				err = conn.Invoke(ctx, fmt.Sprintf("/%s/%s", svcdesc.FullName(), method.Name()), req, res)
				log.Printf("resp: %v, err: %v\n", res, err)
			}
		}
	}
}
