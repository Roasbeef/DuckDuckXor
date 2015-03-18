package main

import (
	"flag"
	"runtime"

	pb "github.com/roasbeef/DuckDuckXor/protos"
)

var (
	tls = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")

	serverAddr = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")

	documentDirectory = flag.String("doc_dir", ".", "Directory where documents to be indexed live")
)

func init() {
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

}

type clientDaemon struct {
}

func (c *clientDaemon) requestSearch(qstream pb.EncryptedSearch_UploadTSetServer, query string) {
	/*conn, err := grpc.Dial(*serverAddr)
	if err != nil {
		//TODO handle error

	}
	client := pb.NewEncryptedSearchClient(conn)
	//TODO encrypt keyword before query
	client.KeywordSearch(query)*/
}
