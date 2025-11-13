package main

import (
	proto "ITUServer/grpc"
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
)

type ITU_databaceServer struct {
	proto.UnimplementedITUdatabaceServer
	students []string
}

func (s *ITU_databaceServer) GetStudents(ctx context.Context, in *proto.Empty) (*proto.Students, error) {
	return &proto.Students{Students: s.students}, nil
}

func main() {
	server := &ITU_databaceServer{students: []string{}}
	server.students = append(server.students, "John")
	server.students = append(server.students, "Joh")
	server.students = append(server.students, "Jeb")
	server.students = append(server.students, "Jaber")

	server.start_server()
}

func (s *ITU_databaceServer) start_server() {
	grpcserver := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("Server did not work")
	}

	proto.RegisterITUdatabaceServer(grpcserver, s)

	err = grpcserver.Serve(listener)
	if err != nil {
		log.Fatalf("this did not work")
	}
}
