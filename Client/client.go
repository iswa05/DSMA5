package main

import (
	proto "Replica/grpc"
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect")
	}

	client := proto.NewITUdatabaceClient(conn)

	students, err := client.GetStudents(context.Background(), &proto.Empty{})
	if err != nil {
		log.Fatalf("we recived nothing or failded to send")
	}

	for _, student := range students.Students {
		log.Println(" - " + student)
	}
}
