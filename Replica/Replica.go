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
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuctionReplica struct {
	proto.UnimplementedReplicaServer
}

var id int32
var isLeader bool
var auctionDuration int = 20

type Auction struct {
	clientId   int32
	highestBid int32
	isOver     bool
}

type AuctionManager struct {
	sync.Mutex
	auctions []Auction
}

var manager AuctionManager
var otherReplica proto.ReplicaClient

func main() {
	id = readIdFromUser()
	server := &AuctionReplica{}

	connectToOtherReplica()
	server.startServer()
}

func AuctionTimer() {
	time.Sleep(time.Second * time.Duration(auctionDuration))

	manager.Lock()
	manager.auctions[len(manager.auctions)-1].isOver = true

	log.Println("Auction is over and bidClient", manager.auctions[len(manager.auctions)-1].clientId, "won with", manager.auctions[len(manager.auctions)-1].highestBid)

	manager.Unlock()
}

func (s *AuctionReplica) Bid(ctx context.Context, bid *proto.Bid) (*proto.Ack, error) {
	if isLeader {
		return handleBid(ctx, bid)
	}

	ack, err := otherReplica.Bid(ctx, bid)
	if err != nil {
		isLeader = true
		return handleBid(ctx, bid)
	}

	return ack, nil
}

func handleBid(ctx context.Context, bid *proto.Bid) (*proto.Ack, error) {
	manager.Lock()

	// create new Auction if no action is in progres
	lengthOfAuctions := len(manager.auctions)
	if lengthOfAuctions < 1 || manager.auctions[lengthOfAuctions-1].isOver {
		auction := &Auction{
			clientId:   bid.GetClientId(),
			highestBid: bid.GetAmount(),
			isOver:     false}
		manager.auctions = append(manager.auctions, *auction)

		manager.Unlock()
		go AuctionTimer()
		SyncReplica(ctx, bid)

		return &proto.Ack{Outcome: "success"}, nil
	}

	// update Auction
	if manager.auctions[lengthOfAuctions-1].highestBid < bid.GetAmount() {

		manager.auctions[lengthOfAuctions-1].clientId = bid.GetClientId()
		manager.auctions[lengthOfAuctions-1].highestBid = bid.GetAmount()

		manager.Unlock()
		SyncReplica(ctx, bid)
		return &proto.Ack{Outcome: "success"}, nil
	} else {
		manager.Unlock()
		return &proto.Ack{Outcome: "fail"}, nil
	}
}

func (s *AuctionReplica) Result(ctx context.Context, request *proto.Empty) (*proto.Result, error) {
	// handle result request
	if isLeader {
		return HandleResult()
	}

	res, err := otherReplica.Result(ctx, request)
	if err != nil {
		isLeader = true
		return HandleResult()
	}

	return res, nil
}

func HandleResult() (*proto.Result, error) {
	manager.Lock()

	lengthOfAuctions := len(manager.auctions)
	log.Println("Auctions:", lengthOfAuctions)
	if lengthOfAuctions == 0 {
		log.Println("No auctions have been held")
		manager.Unlock()
		return &proto.Result{AuctionId: -1}, nil
	}

	var auctionResult = &proto.Result{ClientId: manager.auctions[lengthOfAuctions-1].clientId,
		HighestBid:    manager.auctions[lengthOfAuctions-1].highestBid,
		AuctionIsOver: manager.auctions[lengthOfAuctions-1].isOver,
		AuctionId:     int32(lengthOfAuctions - 1),
	}

	manager.Unlock()
	return auctionResult, nil
}

func (s *AuctionReplica) startServer() {
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

func connectToOtherReplica() {
	var otherReplicaPort int
	if id == 1 {
		otherReplicaPort = 8002
	} else if id == 2 {
		otherReplicaPort = 8001
	}

	otherReplicaPortString := ":" + strconv.Itoa(otherReplicaPort)
	conn, err := grpc.NewClient("localhost"+otherReplicaPortString, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Println("Could not connect to other replica")
	}
	otherReplica = proto.NewReplicaClient(conn)
}

func (s *AuctionReplica) BidMemorySync(ctx context.Context, state *proto.MemoryState) (*proto.Empty, error) {
	log.Println("Syncing auction:", state.AuctionId)

	if isLeader {
		return &proto.Empty{}, nil
	}

	manager.Lock()

	auction := &Auction{
		clientId:   state.GetAuctionClientId(),
		highestBid: state.GetAuctionHighestBid(),
		isOver:     state.GetAuctionIsOver(),
	}

	var auctionLength = int32(len(manager.auctions))
	var stateAuctionId = state.GetAuctionId()

	// If the received auction already has started
	if auctionLength > stateAuctionId {
		manager.auctions[stateAuctionId] = *auction
		manager.Unlock()
		return &proto.Empty{}, nil
	}

	// If the received auction is a new auction
	// There may be possible issues where a new auction sync is delayed and another new auction sync overtakes
	manager.auctions = append(manager.auctions, *auction)
	go AuctionTimer()
	manager.Unlock()

	return &proto.Empty{}, nil
}

func SyncReplica(ctx context.Context, bid *proto.Bid) {
	log.Println("SyncReplica called")
	manager.Lock()
	var lengthOfAuctions = len(manager.auctions)

	state := &proto.MemoryState{
		AuctionId:         int32(lengthOfAuctions - 1),
		AuctionClientId:   bid.GetClientId(),
		AuctionHighestBid: bid.GetAmount(),
		AuctionIsOver:     manager.auctions[lengthOfAuctions-1].isOver,
	}
	manager.Unlock()

	log.Println("State saved")

	_, err := otherReplica.BidMemorySync(ctx, state)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	log.Println("Auctions synced")
}
