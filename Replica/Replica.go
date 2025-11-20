package main

import (
	proto "Replica/grpc"
	"bufio"
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc"
)

type AuctionReplica struct {
	proto.UnimplementedReplicaServer
}

var id int32
var isLeader bool
var auctionDuration int = 20

type auction struct {
	clientId   int32
	highestBid int32
	isOver     bool
}

var auctions []auction

func main() {
	id = readIdFromUser()
	server := &AuctionReplica{}

	server.start_server()
}

func (s *AuctionReplica) Bid(ctx context.Context, bid *proto.Bid) (*proto.Ack, error) {

	// create new auction if no action is in progres
	lengOfAuctions := len(auctions)
	if lengOfAuctions < 1 || auctions[lengOfAuctions-1].isOver {

		auction := &auction{
			clientId:   bid.GetClientId(),
			highestBid: bid.GetAmount(),
			isOver:     false}
		auctions = append(auctions, *auction)

		// start new timer
	}

	// update auction

	return &proto.Ack{}, nil
}

func (s *AuctionReplica) Result(ctx context.Context, request *proto.Empty) (*proto.Result, error) {
	// handle result request

	return &proto.Result{}, nil
}

func (s *AuctionReplica) start_server() {
	grpcserver := grpc.NewServer()
	port := ":" + strconv.Itoa(int(8000+id))
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Server did not work")
	}

	proto.RegisterReplicaServer(grpcserver, s)

	err = grpcserver.Serve(listener)
	if err != nil {
		log.Fatalf("this did not work")
	}
}

func (s *AuctionReplica) Ping(ctx context.Context, request *proto.Empty) (*proto.Empty, error) {
	return &proto.Empty{}, nil
}

func readIdFromUser() int32 {
	var inputInt int
	log.Println("Please enter a unique id, either 1 or 2")
	for {
		var err error
		inputString := readFromUser()
		inputInt, err = strconv.Atoi(inputString)
		if err != nil {
			log.Println("Invalid input type, please enter a valid integer")
			continue
		}

		if inputInt == 1 || inputInt == 2 {
			if inputInt == 1 {
				isLeader = true
			}
			return int32(inputInt)
		}

		log.Println("Invalid id value, must be either 1 or 2")
	}

}

func readFromUser() string {
	reader := bufio.NewReader(os.Stdin)
	inputString, _ := reader.ReadString('\n')
	inputString = strings.TrimSuffix(inputString, "\n")
	inputString = strings.TrimSuffix(inputString, "\r")

	return inputString
}
