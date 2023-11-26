package main

import (
	proto "bidding/grpc"
	"bufio"
	"context"
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	port       = flag.String("port", "", "server port number")
	input      = bufio.NewScanner(os.Stdin)
	name       string
	client     proto.BiddingClient
	highestBid = &proto.Amount{Amount: 0}
)

func ConnectToServer() {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(":"+*port, opts...)
	checkError(err)

	client = proto.NewBiddingClient(conn)

	cprint("the connection is: " + conn.GetState().String())
}

func getHighestBid() {
	for {
		time.Sleep(time.Second)
		ack, err := client.Result(context.Background(), &proto.Void{})
		if err != nil {
			println("Server has crashed, reconnecting...")
			time.Sleep(5 * time.Second)
			ConnectToServer()
		} else {
			handleHighestBid(ack)
		}

	}
}

func handleHighestBid(ack *proto.Outcome) {
	if !ack.IsResult && highestBid.Amount < ack.HighestBid.Amount {
		cprint("NEW HIGHEST BID!! " + ack.HighestBid.Name + " has bid " + strconv.Itoa(int(ack.HighestBid.Amount)) + "!!")
		highestBid = ack.HighestBid
	}
}

func main() {
	flag.Parse()

	println("Please input name: ")
	input.Scan()
	name = strings.TrimSpace(input.Text())

	ConnectToServer()
	/*
		initAmount := &proto.Amount{
			Amount: 0,
			Name:   name,
		}

		ack, err := client.Bid(context.Background(), initAmount)
		checkError(err)
		cprint(string(ack.HighestBid))
	*/
	go getHighestBid()
	for {
		input.Scan()
		inputWords := strings.Split(input.Text(), " ")
		if inputWords[0] == "bid" {
			amt, err := strconv.Atoi(inputWords[1])
			checkError(err)
			if int(highestBid.Amount) >= amt {
				cprint("Bid not high enough! " + highestBid.Name + " has bid " + strconv.Itoa(int(highestBid.Amount)) + "!")
			} else {
				bid := &proto.Amount{
					Amount: int64(amt),
					Name:   name,
				}
				_, err := client.Bid(context.Background(), bid)
				checkError(err)
			}

		} else if inputWords[0] == "exit" {
			cprint("Quitting bidding session...")
			return
		} else {
			cprint("Unrecognized command")
		}
	}
}

func cprint(s string) {
	if s != "" {
		println(s)
	}
	print("->")
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
