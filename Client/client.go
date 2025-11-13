package main

import (
	proto "Replica/grpc"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect")
	}

	client := proto.NewReplicaClient(conn)

	if client != nil {
		log.Fatal("Oh no!")
	}
}
