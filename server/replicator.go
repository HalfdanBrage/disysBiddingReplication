package main

import (
	proto "bidding/grpc"
	"context"
	"flag"
	"io"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Replicator struct {
	proto.UnimplementedBiddingServer
	id int
	port string
}


var port = flag.String("port", "", "server port number")
var role = flag.String("role", "", "replicator role")

func (r *Replicator) ConnectToReplicator(connectRequest *proto.ConnectRequest, stream proto.Bidding_ConnectToReplicatorServer) error {

	bid1 := &proto.ReplicatorUpdate{
		Bidder: "bidder1",
	}

	bid2 := &proto.ReplicatorUpdate{
		Bidder: "bidder2",
	}

	stream.Send(bid1)

	time.Sleep(time.Second)

	stream.Send(bid2)

	return nil
}


//When setting a secondary replicator as primary remember to stop secondary processes
func setupAsPrimary() {
	replicator := &Replicator{
		id: 0,
		port: *port,
	}

	go startServer(replicator)
}

func startServer(replicator *Replicator) {
	grpcServer := grpc.NewServer()

	listener, err := net.Listen("tcp", ":" + *port)
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

	updateStream, err := primaryConn.ConnectToReplicator(context.Background(), &proto.ConnectRequest{})
	checkError(err)
	for {
		update, err := updateStream.Recv()
		if err == io.EOF {
			break
		}
		checkError(err)
		log.Println(update.Bidder)
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
