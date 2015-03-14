package main

import (
	"flag"
	"fmt"
	"io"
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
	dbFile   = flag.String("db_file", "~/.duckduckxor/db.bolt", "Location of DB file")
)

// encryptedSearchServer....
type encryptedSearchServer struct {
	docStore *documentDatabase
	searcher *encryptedIndexSearcher

	metaDataRecived chan struct{}
}

// UploadMetaData loads the meta data required for creating and searching
// through the encrypted index.
func (e *encryptedSearchServer) UploadMetaData(ctx context.Context, mData *pb.MetaData) (*pb.MetaDataAck, error) {
	e.searcher.LoadTSetMetaData(mData)
	close(e.metaDataRecived)
	return &pb.MetaDataAck{Ack: true}, nil
}

func (e *encryptedSearchServer) WaitForMetaDataInit() {
	<-e.metaDataRecived
}

func (e *encryptedSearchServer) UploadTSet(stream pb.EncryptedSearch_UploadTSetServer) error {
	e.WaitForMetaDataInit()
	return nil
}

func (e *encryptedSearchServer) UploadXSetFilter(ctx context.Context, xFilter *pb.XSetFilter) (*pb.FilterAck, error) {
	return nil, nil
}

// UploadCipherDocs implements a client streaming RPC for storing encrypted
// documents on the server.
func (e *encryptedSearchServer) UploadCipherDocs(stream pb.EncryptedSearch_UploadCipherDocsServer) error {
	for {
		doc, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.CipherDocAck{
				Ack: true,
			})
		}

		e.docStore.PutDoc(doc)
	}
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

func newEncryptedSearchServer(docStore *documentDatabase) (*encryptedSearchServer, error) {
	return &encryptedSearchServer{
		metaDataRecived: make(chan struct{}),
		docStore:        docStore,
	}, nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Open up our database, creating the file if needed.
	db, err := bolt.Open(*dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize components for the server.
	grpcServer := grpc.NewServer()
	docDb, err := NewDocumentDatabase(db)
	if err != nil {
		log.Fatalf("unable to create document database: %v", err)
	}
	docDb.Start()

	eServer, err := newEncryptedSearchServer(docDb)
	if err != nil {
		log.Fatalf("unable to create server: %v", err)
	}

	// Register our implemented server interface
	pb.RegisterEncryptedSearchServer(grpcServer, eServer)

	// Optionally activate TLS.
	if *tls {
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		grpcServer.Serve(creds.NewListener(lis))
	} else {
		grpcServer.Serve(lis)
	}

	// TODO(roasbeef): OS Signal/Error handling.
}
