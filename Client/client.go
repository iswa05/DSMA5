package main

import (
	proto "Replica/grpc"
	"bufio"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var id int32
var portId int

var ports = []string{":8001", ":8002"}

func main() {
	id = readIdFromUser()

	portId = rand.Intn(2)
	conn, err := connectToServers()
	if err != nil {
		log.Fatal("Could not connect to any")
	}

	client := proto.NewReplicaClient(conn)

	if client != nil {
		log.Fatal("Oh no!")
	}
}

func connectToServers() (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 2; i++ {
		log.Println("Trying to connect to " + ports[portId])
		conn, err = grpc.NewClient("localhost:"+ports[portId], grpc.WithTransportCredentials(insecure.NewCredentials()))

		if err == nil {
			return conn, nil
		}
		log.Println("could not connect to " + ports[portId])
		portId = (portId + 1) % 2
	}

	return conn, err
}

func readIdFromUser() int32 {
	var inputInt int
	log.Println("Please enter a unique id")
	for {
		var err error
		inputString := readFromUser()
		inputInt, err = strconv.Atoi(inputString)
		if err != nil {
			log.Println("Invalid input type, please enter a valid integer")
			continue
		}

		if inputInt > 0 {
			return int32(inputInt)
		}

		log.Println("Invalid id value, must be positive")
	}

}

func readFromUser() string {
	reader := bufio.NewReader(os.Stdin)
	inputString, _ := reader.ReadString('\n')
	inputString = strings.TrimSuffix(inputString, "\n")
	inputString = strings.TrimSuffix(inputString, "\r")

	return inputString
}
