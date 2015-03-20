package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"

	"github.com/boltdb/bolt"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var dbPath = "/Users/dimberman/.duckduckxor/"

var (
	port     = flag.Int("port", 10000, "The server port")
	tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile = flag.String("cert_file", "testdata/server1.pem", "The TLS cert file")
	keyFile  = flag.String("key_file", "testdata/server1.key", "The TLS key file")
	dbFile   = flag.String("db_file", dbPath+"ddx.db", "Location of DB file")
)

// encryptedSearchServer....
type encryptedSearchServer struct {
	docStore       *documentDatabase
	encryptedIndex *encryptedIndexSearcher
}

// UploadMetaData loads the meta data required for creating and searching
// through the encrypted index.
func (e *encryptedSearchServer) UploadMetaData(ctx context.Context, mData *pb.MetaData) (*pb.MetaDataAck, error) {
	e.encryptedIndex.LoadTSetMetaData(mData)
	return &pb.MetaDataAck{Ack: true}, nil
}

func (e *encryptedSearchServer) UploadTSet(stream pb.EncryptedSearch_UploadTSetServer) error {
	fmt.Println("got uplaod tset req")
	for {
		tTuple, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.TSetAck{
				Ack: true,
			})
		}

		fmt.Println("putting frag serv", tTuple)
		e.encryptedIndex.PutTsetFragment(tTuple)
	}
}

// UploadXSetFilter loads the client-side created xSet bloom filter into memory
// and the database.
func (e *encryptedSearchServer) UploadXSetFilter(ctx context.Context, xFilter *pb.XSetFilter) (*pb.FilterAck, error) {
	fmt.Println("got uplaod xset, ", xFilter)
	e.encryptedIndex.PutXSetFilter(xFilter)
	return &pb.FilterAck{Ack: true}, nil
}

// UploadCipherDocs implements a client streaming RPC for storing encrypted
// documents on the server.
func (e *encryptedSearchServer) UploadCipherDocs(stream pb.EncryptedSearch_UploadCipherDocsServer) error {
	fmt.Println("reading cipher docs")
	for {
		doc, err := stream.Recv()
		if err == io.EOF || doc == nil {
			// TODO(Roasbeef): Proper closure?
			return stream.SendAndClose(&pb.CipherDocAck{
				Ack: true,
			})
		}

		fmt.Println("putting doc", doc)
		e.docStore.PutDoc(doc)
	}
}

func (e *encryptedSearchServer) KeywordSearch(query *pb.KeywordQuery, stream pb.EncryptedSearch_KeywordSearchServer) error {
	stag := query.Stag

	// Kick off the single word search.
	respStream, cancelSearch := e.encryptedIndex.TSetSearch(stag)

	// Read responses one by one off the channel, then sending them down
	// the client stream.
	for resp := range respStream {
		if err := stream.Send(&pb.EncryptedDocInfo{EncryptedId: resp.eId}); err != nil {
			cancelSearch <- struct{}{}
			return err
		}
	}
	return nil
}

func (e *encryptedSearchServer) FetchDocuments(stream pb.EncryptedSearch_FetchDocumentsServer) error {
	for {
		docReq, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		doc, err := e.docStore.RetrieveDoc(docReq.DocId)
		if err != nil {
			return err
		}

		if err := stream.Send(doc); err != nil {
			return err
		}
	}
	return nil
}

func (e *encryptedSearchServer) ConjunctiveSearchRequest(stream pb.EncryptedSearch_ConjunctiveSearchRequestServer) error {
	return nil
}

func (e *encryptedSearchServer) XTokenExchange(stream pb.EncryptedSearch_XTokenExchangeServer) error {
	return nil
}

func newEncryptedSearchServer(docStore *documentDatabase, index *encryptedIndexSearcher) (*encryptedSearchServer, error) {
	return &encryptedSearchServer{
		docStore:       docStore,
		encryptedIndex: index,
	}, nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Create our dir if it doesn't exist.
	if err := os.MkdirAll(dbPath, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Open up our database, creating the file if needed.
	db, err := bolt.Open(*dbFile, os.ModePerm, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize components for the server.
	docDb, err := NewDocumentDatabase(db)
	if err != nil {
		log.Fatalf("unable to create document database: %v", err)
	}
	docDb.Start()

	index, err := NewEncryptedIndexSearcher(db)
	if err != nil {
		log.Fatalf("unable to load search index: %v", err)
	}
	index.Start()

	eServer, err := newEncryptedSearchServer(docDb, index)
	if err != nil {
		log.Fatalf("unable to create server: %v", err)
	}

	// Register our implemented server interface
	grpcServer := grpc.NewServer()
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
