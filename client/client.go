package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"flag"
	"io"
	"log"
	"runtime"

	"github.com/boltdb/bolt"
	"github.com/conformal/btcwallet/snacl"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	tls = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")

	serverAddr = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")

	documentDirectory = flag.String("doc_dir", ".", "Directory where documents to be indexed live")

	doIndex = flag.String("index", "false", "perform indexing if true")
)

func init() {
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	//read config file
	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	c := NewClientDaemon(db)
	if *doIndex == "true" {
		c.Index(*documentDirectory)
	}
	awaitCommands(c)
}

func awaitCommands(c *clientDaemon) {
	for {

	}
}

type plainDoc struct {
	plainbytes []byte
	docId      uint32
}

type clientDaemon struct {
	eDocs     chan *pb.EncryptedDocInfo
	keys      *KeyManager
	docKey    snacl.CryptoKey
	docNames  map[uint32]string
	plainDocs chan plainDoc
}

func NewClientDaemon(db *bolt.DB) *clientDaemon {
	key, err := NewKeyManager(db, nil)
	if err != nil {

	}
	return &clientDaemon{
		eDocs:     make(chan *pb.EncryptedDocInfo),
		keys:      key,
		docNames:  make(map[uint32]string),
		plainDocs: make(chan plainDoc),
	}
}

func (c *clientDaemon) search(query string) {
	c.requestSearch(query)

}

func (c *clientDaemon) Index(root string) {
	i := NewIndexer()
	i.Index(root)
	go func(i *indexer) {

		c.docNames = <-i.nameChannel

	}(i)

}

func (c *clientDaemon) requestSearch(query string) {
	conn, err := grpc.Dial(*serverAddr)
	if err != nil {
		//TODO handle error

	}
	client := pb.NewEncryptedSearchClient(conn)
	//TODO encrypt keyword before query
	kQuery := &pb.KeywordQuery{c.encryptQuery(query)}
	e, err := client.KeywordSearch(context.Background(), kQuery)
	if err != nil {
		//TODO handle error

	}
	c.recieveDocuments(e)
}

func (c *clientDaemon) recieveDocuments(stream pb.EncryptedSearch_KeywordSearchClient) {
	for {
		doc, err := stream.Recv()
		if err == io.EOF {
			break
		}
		c.eDocs <- doc
	}

}

func (c *clientDaemon) sendFetchRequests(client pb.EncryptedSearchClient) {

	fetch, err := client.FetchDocuments(context.Background())
	if err != nil {
		//TODO handle errors
	}
	for {
		select {
		case doc, more := <-c.eDocs:
			if !more {
				break
			}
			err = fetch.Send(decryptDocInfo(doc))
			if err != nil {
				//TODO handle errors
			}

		}
	}

}

func (c *clientDaemon) fetchDocuments(client pb.EncryptedSearchClient) {
	fetch, err := client.FetchDocuments(context.Background())
	if err != nil {
		//TODO handle errors
	}
	for {
		cDoc, err := fetch.Recv() //cipherDoc = uint32 doc_id, bytes encrypted_doc
		//TODO are we recieving a continuous stream/ handling death?
		if err != nil {
			//TODO handle errors
		}
		b := c.decryptDoc(cDoc)
		c.plainDocs <- plainDoc{b, cDoc.DocId}
	}
}

func (c *clientDaemon) encryptQuery(s string) []byte {
	stag := c.keys.FetchSTagKey()
	hm := hmac.New(sha1.New, stag[:16])
	hm.Write([]byte(s))
	return hm.Sum(nil)
}

func decryptDocInfo(eDoc *pb.EncryptedDocInfo) *pb.DocInfo {
	return &pb.DocInfo{0}

}

func (c *clientDaemon) decryptDoc(eDoc *pb.CipherDoc) []byte {
	plainTextBytes, err := c.docKey.Decrypt(eDoc.EncryptedDoc)
	//TODO store doc names
	if err != nil {
		//TODO handle error
	}
	return plainTextBytes
}
