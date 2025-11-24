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
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	id       int32
	ports    = []string{":8001", ":8002"}
	portId   int
	client   proto.ReplicaClient
	lamport  int64
	lamMutex sync.Mutex
)

func main() {
	id = readIdFromUser()
	portId = rand.Intn(len(ports))

	var err error
	client, err = connectToServers()
	if err != nil {
		log.Fatal("Could not connect to any server:", err)
	}

	for {
		log.Println("Enter a command: \n bid <amount> \n result")
		input := readFromUser()
		args := strings.Split(input, " ")
		if len(args) < 1 || len(args) > 2 {
			log.Println("Invalid command format")
			continue
		}

		switch args[0] {
		case "bid":
			if len(args) != 2 {
				log.Println("Usage: bid <amount>")
				continue
			}
			amount, err := strconv.Atoi(args[1])
			if err != nil {
				log.Println("Invalid amount, must be an integer")
				continue
			}
			makeBid(int32(amount))
		case "result":
			getResult()
		default:
			log.Println("Unknown command")
		}
	}
}

func connectToServers() (proto.ReplicaClient, error) {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < len(ports); i++ {
		conn, err = grpc.NewClient("localhost"+ports[portId], grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			c := proto.NewReplicaClient(conn)
			_, err = c.Ping(context.Background(), &proto.Empty{})
			if err == nil {
				return c, nil
			}
		}
		portId = (portId + 1) % len(ports)
	}
	return nil, err
}

func makeBid(amount int32) {
	lam := nextLamport()
	bidMsg := &proto.Bid{ClientId: id, Amount: amount, Lamport: lam}

	ack, err := client.Bid(context.Background(), bidMsg)
	if err != nil {
		client, err = connectToServers()
		if err == nil {
			makeBid(amount)
		}
		return
	}

	// update lamport safely
	lamMutex.Lock()
	if ack.Lamport >= lamport {
		lamport = ack.Lamport + 1
	} else {
		lamport++
	}
	lamMutex.Unlock()

	log.Println("Bid outcome:", ack.Outcome)
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

	status := "ongoing"
	if result.AuctionIsOver {
		status = "over"
	}

	log.Printf("Auction %d is %s | Highest bid: %d from client %d\n",
		result.AuctionId, status, result.HighestBid, result.ClientId)
}

func readIdFromUser() int32 {
	log.Println("Please enter a unique id")
	for {
		input := readFromUser()
		val, err := strconv.Atoi(input)
		if err == nil && val > 0 {
			return int32(val)
		}
		log.Println("Invalid id value, must be a positive integer")
	}
}

func readFromUser() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func nextLamport() int64 {
	lamMutex.Lock()
	defer lamMutex.Unlock()
	lamport++
	return lamport
}
