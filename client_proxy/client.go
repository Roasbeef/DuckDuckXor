package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/conformal/btcwallet/snacl"
	"github.com/jacobsa/crypto/cmac"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	tls = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")

	serverAddr = flag.String("server_addr", "localhost:10000", "The server address in the format of host:port")

	port              = flag.Int("port", 20003, "The client proxy port")
	documentDirectory = flag.String("doc_dir", "hold", "Directory where documents to be indexed live")

	doIndex    = flag.Bool("index", false, "perform indexing if true")
	passPhrase = flag.String("passphrase", "", "password needed to access keys")
	numWorkers = flag.Int("num_workers", 1, "number of workers allowed at a time")
	certFile   = flag.String("cert_file", "testdata/server1.pem", "The TLS cert file")
	keyFile    = flag.String("key_file", "testdata/server1.key", "The TLS key file")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	conn, err := grpc.Dial(*serverAddr)
	if err != nil {
		log.Fatal(err)
	}

	client := pb.NewEncryptedSearchClient(conn)
	metaAck, err := client.UploadMetaData(context.Background(), &pb.MetaData{4, 28})
	if metaAck.Ack != true || err != nil {
		log.Fatal(err)
	}

	//read config file
	db, err := bolt.Open("/Users/dimberman/.duckduckxor/client_ddx.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var key *KeyManager
	var bloom *bloomMaster
	var names map[uint32]string

	//if *doIndex == true {
	if true {
		bloom, key, names = Index(*documentDirectory, db, client, []byte(*passPhrase))
	} else {
		//TODO need to find a way to persistantly store the id->name map
		var wg sync.WaitGroup
		key, err = NewKeyManager(db, []byte(*passPhrase), &wg, nil)
		if err != nil {
			log.Fatal(err)
		}
		bloom, err = newBloomMaster(db, *numWorkers, &wg, nil, client)
		if err != nil {
			log.Fatal(err)
		}
	}

	proxyServer, err := NewClientDaemon(db, key, bloom, names)
	if err != nil {
		log.Fatal(err)
	}

	// Start up the client proxy.
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Register our implemented server interface
	grpcServer := grpc.NewServer()
	pb.RegisterProxySearchServer(grpcServer, proxyServer)

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
}

type clientDaemon struct {
	eDocs     chan *pb.EncryptedDocInfo
	keys      *KeyManager
	bm        *bloomMaster
	docKey    snacl.CryptoKey
	docNames  map[uint32]string
	plainDocs chan *pb.PlainDoc
	client    pb.EncryptedSearchClient
}

func NewClientDaemon(db *bolt.DB, k *KeyManager, b *bloomMaster, dNames map[uint32]string) (*clientDaemon, error) {
	conn, err := grpc.Dial(*serverAddr)
	if err != nil {
		return nil, err
	}
	client := pb.NewEncryptedSearchClient(conn)
	return &clientDaemon{
		eDocs:     make(chan *pb.EncryptedDocInfo),
		keys:      k,
		bm:        b,
		docNames:  dNames,
		plainDocs: make(chan *pb.PlainDoc),
		client:    client,
	}, nil
}

func (c *clientDaemon) Search(query *pb.PlainTextQuery, stream pb.ProxySearch_SearchServer) error {
	word := query.QueryWord

	go c.requestSearch(word)

	go c.sendFetchRequests(c.client)

	for doc := range c.plainDocs {
		stream.Send(doc)
	}

	return nil
}

func Index(root string, db *bolt.DB, client pb.EncryptedSearchClient, pass []byte) (*bloomMaster, *KeyManager, map[uint32]string) {
	i := NewIndexer()
	//	go func(i *indexer) {
	//		c.docNames = <-i.nameChannel
	//	}(i)
	return i.Index(root, db, *numWorkers, client, pass)
}

func (c *clientDaemon) requestSearch(query string) {
	kQuery := &pb.KeywordQuery{c.encryptQuery(query)}
	stream, err := c.client.KeywordSearch(context.Background(), kQuery)
	if err != nil {
		fmt.Println("ERROR CD: %v", err)
	}

	for {
		doc, err := stream.Recv()
		if err == io.EOF {
			close(c.eDocs)
			break
		}
		c.eDocs <- doc
	}
}

func (c *clientDaemon) sendFetchRequests(client pb.EncryptedSearchClient) {

	fetch, err := client.FetchDocuments(context.Background())
	if err != nil {
		//TODO handle errors
		fmt.Println("ERROR FETCH: %v", err)
	}
	for {
		select {
		case doc, more := <-c.eDocs:
			if !more {
				fetch.CloseSend()
				break
			}
			if err := fetch.Send(decryptDocInfo(doc)); err != nil {
				fmt.Println("ERROR FETCH: %v", err)
			}

		}
	}

	go func() {
		for {
			cDoc, err := fetch.Recv() //cipherDoc = uint32 doc_id, bytes encrypted_doc
			if err == io.EOF {
				fmt.Println("ERROR FETCH: %v", err)
				close(c.plainDocs)
			}
			b := c.decryptDoc(cDoc)
			c.plainDocs <- &pb.PlainDoc{DocBytes: b, DocId: cDoc.DocId}
		}
	}()

}

func (c *clientDaemon) encryptQuery(s string) []byte {
	stag := c.keys.FetchSTagKey()
	hm, _ := cmac.New(stag[:])
	hm.Write([]byte(s))
	return hm.Sum(nil)
}

func decryptDocInfo(eDoc *pb.EncryptedDocInfo) *pb.DocInfo {
	//TODO later on this will actually decrypt
	return &pb.DocInfo{binary.BigEndian.Uint32(eDoc.EncryptedId)}
}

func (c *clientDaemon) decryptDoc(eDoc *pb.CipherDoc) []byte {
	plainTextBytes, err := c.docKey.Decrypt(eDoc.EncryptedDoc)
	//TODO store doc names
	if err != nil {
		//TODO handle error
	}
	return plainTextBytes
}
