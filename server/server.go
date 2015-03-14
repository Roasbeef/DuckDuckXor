package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"

	"github.com/boltdb/bolt"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	port     = flag.Int("port", 10000, "The server port")
	tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile = flag.String("cert_file", "testdata/server1.pem", "The TLS cert file")
	keyFile  = flag.String("key_file", "testdata/server1.key", "The TLS key file")
)

type encryptedSearchServer struct {
	docStore *documentDatabase
	db       *bolt.DB
}

func (e *encryptedSearchServer) SendMetaData(ctx context.Context, mData *pb.MetaData) (*pb.MetaDataAck, error) {
	return nil, nil
}

func (e *encryptedSearchServer) UploadTSet(stream pb.EncryptedSearch_UploadTSetServer) error {
	return nil
}

func (e *encryptedSearchServer) UploadXSetFilter(ctx context.Context, xFilter *pb.XSetFilter) (*pb.FilterAck, error) {
	return nil, nil
}

func (e *encryptedSearchServer) UploadCipherDocs(stream pb.EncryptedSearch_UploadCipherDocsServer) error {
	return nil
}

func (e *encryptedSearchServer) KeywordSearch(stream pb.EncryptedSearch_KeywordSearchServer) error {
	return nil
}

func (e *encryptedSearchServer) ConjunctiveSearchRequest(stream pb.EncryptedSearch_ConjunctiveSearchRequestServer) error {
	return nil
}

func (e *encryptedSearchServer) XTokenExchange(stream pb.EncryptedSearch_XTokenExchangeServer) error {
	return nil
}

func newEncryptedSearchServer() (*encryptedSearchServer, error) {
	return nil, nil
}

func init() {
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	eServer, err := newEncryptedSearchServer()
	if err != nil {
		log.Fatalf("unable to create server: %v", err)
	}

	pb.RegisterEncryptedSearchServer(grpcServer, eServer)

	if *tls {
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		grpcServer.Serve(creds.NewListener(lis))
	} else {
		grpcServer.Serve(lis)
	}
}
