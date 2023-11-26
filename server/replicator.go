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
	highestBid         = &proto.Amount{Amount: 0}
	bidUpdateChans     = []chan *proto.ReplicatorUpdate{}
	id             int = 0
	nextId         int
	primaryId      int
)

func (r *Replicator) Bid(ctx context.Context, amount *proto.Amount) (*proto.BidAck, error) {
	println("Recieved bid from: " + amount.Name + " for amount: " + strconv.Itoa(int(amount.Amount)))
	if amount.Amount > highestBid.Amount {
		highestBid = amount
		println("Bid " + strconv.Itoa(int(highestBid.Amount)) + " from " + highestBid.Name + " is now the highest bid")
		for _, replicatorChan := range bidUpdateChans {
			replicatorChan <- &proto.ReplicatorUpdate{
				Bid: highestBid,
			}
		}

	}
	ack := &proto.BidAck{
		HighestBid: highestBid,
	}
	return ack, nil
}

func (r *Replicator) Result(ctx context.Context, _ *proto.Void) (*proto.Outcome, error) {
	out := &proto.Outcome{
		HighestBid: highestBid,
		IsResult:   false,
	}
	return out, nil
}

func (r *Replicator) ConnectToReplicator(connectRequest *proto.Void, stream proto.Bidding_ConnectToReplicatorServer) error {
	replicatorChan := make(chan *proto.ReplicatorUpdate)
	bidUpdateChans = append(bidUpdateChans, replicatorChan)

	setupRu := &proto.ReplicatorUpdate{
		Id:        int64(nextId),
		PrimaryId: int64(id),
		Bid:       highestBid,
	}
	stream.Send(setupRu)
	nextId += 1

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

	sErr := grpcServer.Serve(listener)
	checkError(sErr)
}

func requestPrimaryConnection() {
	conn, err := grpc.Dial("localhost:"+*port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	checkError(err)

	primaryConn := proto.NewBiddingClient(conn)

	updateStream, err := primaryConn.ConnectToReplicator(context.Background(), &proto.Void{})
	checkError(err)
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

	if *role == "primary" {
		setupAsPrimary()
	} else {
		go requestPrimaryConnection()
	}

	for {

	}
}
