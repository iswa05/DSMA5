package main

import (
	proto "Replica/grpc"
	"bufio"
	"context"
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
var client proto.ReplicaClient

func main() {
	id = readIdFromUser()

	portId = rand.Intn(2)
	var err error
	client, err = connectToServers()

	if err != nil {
		log.Fatal("Could not connect to any")
	}

	for {
		log.Println("Enter a commandArgs: \n bid <amount> \n result")
		var userInput = readFromUser()
		var commandArgs = strings.Split(userInput, " ")

		if len(commandArgs) > 2 || len(commandArgs) < 1 {
			log.Println("Invalid commandArgs or commandArgs format")
			continue
		}

		if commandArgs[0] == "bid" && len(commandArgs) == 2 {
			// BID

			var amount, err = strconv.Atoi(commandArgs[1])
			if err != nil {
				log.Println("Invalid amount, must be an integer")
				continue
			}

			makeBid(int32(amount))

		} else if commandArgs[0] == "result" {
			// RESULT
			getResult()

		} else {
			log.Println("Invalid commandArgs or commandArgs format")
			continue
		}
	}

}

func connectToServers() (proto.ReplicaClient, error) {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 2; i++ {
		log.Println("Trying to connect to " + ports[portId])
		conn, err = grpc.NewClient("localhost"+ports[portId], grpc.WithTransportCredentials(insecure.NewCredentials()))
		client := proto.NewReplicaClient(conn)

		_, err := client.Ping(context.Background(), &proto.Empty{})

		if err == nil {
			return client, nil
		}

		log.Println("could not connect to " + ports[portId])
		portId = (portId + 1) % 2
	}

	log.Fatalln("All servers are down")
	return nil, err
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

func makeBid(amount int32) {
	bid, err := client.Bid(context.Background(), &proto.Bid{ClientId: id, Amount: amount})
	if err != nil {
		client, err = connectToServers()

		if err == nil {
			makeBid(amount)
		}
		return
	}
	log.Println("Bid outcome was:", bid.Outcome)
}

func getResult() {
	result, err := client.Result(context.Background(), &proto.Empty{})
	if err != nil {
		client, err = connectToServers()
		if err == nil {
			getResult()
		}
		return
	}
	if result.AuctionId == -1 {
		log.Println("No auctions have been held")
		return
	}

	if result.AuctionIsOver {
		log.Println("Auction", result.AuctionId, "is over with client", result.ClientId, "with the winning bet of", result.HighestBid)
	} else {
		log.Println("Auction", result.AuctionId, "is ongoing with highest bet of", result.HighestBid, "from client", result.ClientId)
	}
}
