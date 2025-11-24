package main

import (
	proto "Replica/grpc"
	"bufio"
	"context"
	"fmt"
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

var (
	id                     int32
	isLeader               bool
	auctionDuration        = 20
	manager                = &AuctionManager{}
	otherReplica           proto.ReplicaClient
	processedMu            sync.Mutex
	processedRequests      = make(map[string]*proto.Ack) // key: clientId-lamport
	auctionTimers          = make(map[int32]*time.Timer)
	timerMutex             sync.Mutex
	serverLamport          int64
	serverLamportMutex     sync.Mutex
	activeAuctionOperation sync.Mutex
)

type Auction struct {
	clientId   int32
	highestBid int32
	isOver     bool
}

type AuctionManager struct {
	sync.Mutex
	auctions []Auction
}

func main() {
	id = readIdFromUser()
	connectToOtherReplica()
	server := &AuctionReplica{}
	server.startServer()
}

func updateLamport(incomingRequest int64) int64 {
	serverLamportMutex.Lock()
	defer serverLamportMutex.Unlock()
	serverLamport = maxInt(serverLamport, incomingRequest) + 1
	return serverLamport
}

func (s *AuctionReplica) startServer() {
	grpcServer := grpc.NewServer()
	port := ":" + strconv.Itoa(8000+int(id))
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("Server listen error:", err)
	}
	proto.RegisterReplicaServer(grpcServer, s)
	log.Println("Server running on", port)
	log.Fatal(grpcServer.Serve(listener))
}

func connectToOtherReplica() {
	var otherPort int
	if id == 1 {
		otherPort = 8002
	} else {
		otherPort = 8001
	}
	conn, err := grpc.NewClient("localhost:"+strconv.Itoa(otherPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Println("Failed to connect to other replica:", err)
		return
	}
	otherReplica = proto.NewReplicaClient(conn)
}

func (s *AuctionReplica) Bid(ctx context.Context, bid *proto.Bid) (*proto.Ack, error) {
	activeAuctionOperation.Lock()
	defer activeAuctionOperation.Unlock()

	if !isLeader && otherReplica != nil {
		ack, err := otherReplica.Bid(ctx, bid)
		if err == nil {
			return ack, nil
		}
		isLeader = true
		log.Printf("Replica %d became leader\n", id)
	}

	serverTs := updateLamport(bid.Lamport)

	key := fmt.Sprintf("%d-%d", bid.ClientId, bid.Lamport)
	processedMu.Lock()
	if ack, exists := processedRequests[key]; exists {
		processedMu.Unlock()
		return ack, nil
	}
	processedMu.Unlock()

	outcome, auctionId := handleBid(bid)
	ack := &proto.Ack{Outcome: outcome, Lamport: serverTs}

	// Sync replica
	SyncReplica(ctx, bid, outcome, auctionId, serverTs)

	processedMu.Lock()
	processedRequests[key] = ack
	processedMu.Unlock()
	return ack, nil
}

func handleBid(bid *proto.Bid) (outcome string, auctionId int32) {
	manager.Lock()
	defer manager.Unlock()
	length := len(manager.auctions)

	if length == 0 || manager.auctions[length-1].isOver {
		manager.auctions = append(manager.auctions, Auction{
			clientId:   bid.ClientId,
			highestBid: bid.Amount,
			isOver:     false,
		})
		auctionId = int32(length)
		go startAuctionTimer(auctionId)
		return "success", auctionId
	}

	if bid.Amount > manager.auctions[length-1].highestBid {
		manager.auctions[length-1].clientId = bid.ClientId
		manager.auctions[length-1].highestBid = bid.Amount
		return "success", int32(length - 1)
	}
	return "fail", int32(length - 1)
}

func (s *AuctionReplica) Result(ctx context.Context, _ *proto.Empty) (*proto.Result, error) {
	activeAuctionOperation.Lock()
	defer activeAuctionOperation.Unlock()

	if !isLeader && otherReplica != nil {
		res, err := otherReplica.Result(ctx, &proto.Empty{})
		if err == nil {
			return res, nil
		}
		isLeader = true
		log.Printf("Replica %d became leader\n", id)
	}

	manager.Lock()
	defer manager.Unlock()
	length := len(manager.auctions)
	if length == 0 {
		return &proto.Result{AuctionId: -1}, nil
	}
	last := manager.auctions[length-1]
	return &proto.Result{
		ClientId:      last.clientId,
		HighestBid:    last.highestBid,
		AuctionIsOver: last.isOver,
		AuctionId:     int32(length - 1),
	}, nil
}

func (s *AuctionReplica) Ping(_ context.Context, _ *proto.Empty) (*proto.Empty, error) {
	return &proto.Empty{}, nil
}

func startAuctionTimer(auctionId int32) {
	timerMutex.Lock()
	if t, exists := auctionTimers[auctionId]; exists {
		t.Stop()
	}
	timer := time.AfterFunc(time.Second*time.Duration(auctionDuration), func() {
		manager.Lock()
		if auctionId < int32(len(manager.auctions)) {
			manager.auctions[auctionId].isOver = true
			log.Printf("Auction %d over, client %d won with %d",
				auctionId, manager.auctions[auctionId].clientId, manager.auctions[auctionId].highestBid)
		}
		manager.Unlock()

		timerMutex.Lock()
		delete(auctionTimers, auctionId)
		timerMutex.Unlock()
	})
	auctionTimers[auctionId] = timer
	timerMutex.Unlock()
}

func (s *AuctionReplica) BidMemorySync(_ context.Context, state *proto.MemoryState) (*proto.Empty, error) {
	if isLeader {
		return &proto.Empty{}, nil
	}

	log.Printf("Backup replica %d received sync for auction %d from leader\n", id, state.AuctionId)
	manager.Lock()
	if int(state.AuctionId) < len(manager.auctions) {
		manager.auctions[state.AuctionId] = Auction{
			clientId:   state.AuctionClientId,
			highestBid: state.AuctionHighestBid,
			isOver:     state.AuctionIsOver,
		}
	} else {
		manager.auctions = append(manager.auctions, Auction{
			clientId:   state.AuctionClientId,
			highestBid: state.AuctionHighestBid,
			isOver:     state.AuctionIsOver,
		})
	}
	manager.Unlock()

	key := fmt.Sprintf("%d-%d", state.AuctionClientId, state.Lamport)
	processedMu.Lock()
	processedRequests[key] = &proto.Ack{Outcome: state.ResponseOutcome, Lamport: state.Lamport}
	processedMu.Unlock()

	if !state.AuctionIsOver {
		go startAuctionTimer(state.AuctionId)
	}
	return &proto.Empty{}, nil
}

func SyncReplica(ctx context.Context, bid *proto.Bid, outcome string, auctionId int32, lamport int64) {
	manager.Lock()
	defer manager.Unlock()

	state := &proto.MemoryState{
		AuctionId:         auctionId,
		AuctionClientId:   bid.ClientId,
		AuctionHighestBid: bid.Amount,
		AuctionIsOver:     manager.auctions[auctionId].isOver,
		Lamport:           lamport,
		ResponseOutcome:   outcome,
	}

	if otherReplica != nil {
		_, err := otherReplica.BidMemorySync(ctx, state)
		if err != nil {
			log.Println("Replica sync failed:", err)
		} else {
			otherId := 3 - id
			log.Printf("Leader (replica %d) completed sync to backup (replica %d) for auction %d\n",
				id, otherId, auctionId)
		}
	}
}

func maxInt(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func readIdFromUser() int32 {
	log.Println("Enter unique replica id (1 or 2)")
	for {
		input := readFromUser()
		val, err := strconv.Atoi(input)
		if err == nil && (val == 1 || val == 2) {
			if val == 1 {
				isLeader = true
			}
			return int32(val)
		}
		log.Println("Invalid ID")
	}
}

func readFromUser() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
