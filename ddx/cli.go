package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	pb "github.com/roasbeef/DuckDuckXor/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	query = flag.String
)

func main() {
	proxyAddr := "localhost:20003"

	conn, err := grpc.Dial(proxyAddr)
	if err != nil {
		log.Fatal(err)
	}

	proxyClient := pb.NewProxySearchClient(conn)
	query := os.Args[2]

	stream, err := proxyClient.Search(context.Background(), &pb.PlainTextQuery{query})

	for {
		doc, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("Got EOF finishing up")
			return
		}
		fmt.Println(doc)
	}
}
