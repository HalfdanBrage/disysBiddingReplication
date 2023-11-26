package main

import (
	proto "bidding/grpc"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Replicator struct {
	proto.UnimplementedBiddingServer
	id   int
	port string
}

var (
	port               = flag.String("port", "", "server port number")
	role               = flag.String("role", "", "replicator role")
	_auctionTime       = flag.Int("time", 60, "Time of the auction in seconds")
	auctionTime    int = 60
	highestBid         = &proto.Amount{Amount: 0}
	bidUpdateChans     = []chan *proto.ReplicatorUpdate{}
	id             int = 0
	nextId         int
	primaryId      int
	timerStarted   bool = false
	mute           sync.Mutex
	isActive       bool = true
)

func (r *Replicator) Bid(ctx context.Context, amount *proto.Amount) (*proto.BidAck, error) {
	println("Recieved bid from: " + amount.Name + " for amount: " + strconv.Itoa(int(amount.Amount)))
	mute.Lock()
	if amount.Amount > highestBid.Amount {
		highestBid = amount
		println("Bid " + strconv.Itoa(int(highestBid.Amount)) + " from " + highestBid.Name + " is now the highest bid")
		for _, replicatorChan := range bidUpdateChans {
			replicatorChan <- &proto.ReplicatorUpdate{
				TimeLeft: int64(auctionTime),
				Bid:      highestBid,
			}
		}

	}
	mute.Unlock()
	ack := &proto.BidAck{
		HighestBid: highestBid,
	}
	return ack, nil
}

func (r *Replicator) Result(ctx context.Context, _ *proto.Void) (*proto.Outcome, error) {
	out := &proto.Outcome{
		TimeLeft:   int64(auctionTime),
		HighestBid: highestBid,
		IsResult:   auctionTime == 0,
	}

	return out, nil
}

func (r *Replicator) ConnectToReplicator(connectRequest *proto.Void, stream proto.Bidding_ConnectToReplicatorServer) error {
	replicatorChan := make(chan *proto.ReplicatorUpdate)
	bidUpdateChans = append(bidUpdateChans, replicatorChan)

	mute.Lock()
	setupRu := &proto.ReplicatorUpdate{
		TimeLeft:  int64(auctionTime),
		Id:        int64(nextId),
		PrimaryId: int64(id),
		Bid:       highestBid,
	}
	stream.Send(setupRu)
	nextId += 1
	mute.Unlock()

	for {
		ru := <-replicatorChan
		stream.Send(ru)
	}
}

// When setting a secondary replicator as primary remember to stop secondary processes
func setupAsPrimary() {
	replicator := &Replicator{
		id:   0,
		port: *port,
	}
	if id == 0 {
		id = 1
	}
	nextId = id + 1

	go startServer(replicator)
}

func startServer(replicator *Replicator) {
	grpcServer := grpc.NewServer()

	listener, err := net.Listen("tcp", ":"+*port)
	checkError(err)
	log.Println("Primary replicator started on port " + *port)

	proto.RegisterBiddingServer(grpcServer, replicator)

	go updateTime()

	sErr := grpcServer.Serve(listener)
	checkError(sErr)

}

func updateTime() {
	if !timerStarted {
		timerStarted = true
		for auctionTime > 0 {
			time.Sleep(time.Second)
			if auctionTime%10 == 0 {
				println(fmt.Sprint(auctionTime) + " seconds left of the auction")
			}
			auctionTime--
		}

		println("The auction is finished")
		time.Sleep(time.Second * 5)
		isActive = false
	}
}

func requestPrimaryConnection() {
	conn, err := grpc.Dial("localhost:"+*port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	checkError(err)

	primaryConn := proto.NewBiddingClient(conn)

	updateStream, err := primaryConn.ConnectToReplicator(context.Background(), &proto.Void{})
	checkError(err)
	go updateTime()
	for {
		update, err := updateStream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			println("Primary replicator crashed! your id: " + fmt.Sprint(id) + " thier id: " + fmt.Sprint(primaryId))
			time.Sleep(time.Second * 2)
			conn.Close()
			if id == primaryId+1 {
				println("Setting up as primary")
				setupAsPrimary()
			} else {
				time.Sleep(time.Second * 2)
				go requestPrimaryConnection()
			}
			return
		}
		if update.Id != 0 {
			id = int(update.Id)
			primaryId = int(update.PrimaryId)
			println("Secondary replicator has id: " + fmt.Sprint(id))
		}
		highestBid = update.Bid
		auctionTime = int(update.TimeLeft)
		log.Println("Recieved replicator update with highest bid " + update.Bid.Name + " " + strconv.Itoa(int(update.Bid.Amount)))

	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()
	auctionTime = *_auctionTime

	if *role == "primary" {
		setupAsPrimary()
	} else {
		go requestPrimaryConnection()
	}

	for isActive {

	}
}
