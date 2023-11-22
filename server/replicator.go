package main

import (
	proto "bidding/grpc"
	"context"
	"flag"
	"io"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Replicator struct {
	proto.UnimplementedBiddingServer
	id   int
	port string
}

var (
	port           = flag.String("port", "", "server port number")
	role           = flag.String("role", "", "replicator role")
	highestBid     = &proto.Amount{Amount: 0}
	bidUpdateChans = []chan *proto.ReplicatorUpdate{}
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
		}
		checkError(err)
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
