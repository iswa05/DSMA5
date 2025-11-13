package main

import (
	proto "Replica/grpc"
	"log"
	"net"

	"google.golang.org/grpc"
)

type AuctionReplica struct {
	proto.UnimplementedReplicaServer
}

func main() {
	server := &AuctionReplica{}

	server.start_server()
}

func (s *AuctionReplica) start_server() {
	grpcserver := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("Server did not work")
	}

	proto.RegisterReplicaServer(grpcserver, s)

	err = grpcserver.Serve(listener)
	if err != nil {
		log.Fatalf("this did not work")
	}
}
