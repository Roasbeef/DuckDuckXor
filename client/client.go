package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"runtime"
	"sync"

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

	doIndex    = flag.Bool("index", false, "perform indexing if true")
	passPhrase = flag.String("passPhrase", "", "password needed to access keys")
	numWorkers = flag.Int("numWorkers", 1, "number of workers allowed at a time")
)

func init() {
	//c := NewCLI()
	//c.Start()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	conn, err := grpc.Dial(*serverAddr)
	if err != nil {
		//TODO handle error

	}
	client := pb.NewEncryptedSearchClient(conn)
	metaAck, err := client.UploadMetaData(context.Background(), &pb.MetaData{4, 28})
	if metaAck.Ack != true || err != nil {

	}
	//read config file
	db, err := bolt.Open("~/.DuckDuckXor/ddx.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var key *KeyManager
	var bloom *bloomMaster
	var names map[uint32]string
	if *doIndex == true {
		bloom, key, names = Index(*documentDirectory, db)
	} else {
		//TODO need to find a way to persistantly store the id->name map
		var wg sync.WaitGroup
		key, err = NewKeyManager(db, []byte(*passPhrase), &wg)
		if err != nil {
			log.Fatal(err)
		}
		bloom, err = newBloomMaster(db, *numWorkers, &wg)
		if err != nil {
			log.Fatal(err)
		}
	}
	c := NewClientDaemon(db, key, bloom, names)
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
	bm        *bloomMaster
	docKey    snacl.CryptoKey
	docNames  map[uint32]string
	plainDocs chan plainDoc
	client    pb.EncryptedSearchClient
}

func NewClientDaemon(db *bolt.DB, k *KeyManager, b *bloomMaster, dNames map[uint32]string) *clientDaemon {
	conn, err := grpc.Dial(*serverAddr)
	if err != nil {
		//TODO handle error

	}
	client := pb.NewEncryptedSearchClient(conn)
	return &clientDaemon{
		eDocs:     make(chan *pb.EncryptedDocInfo),
		keys:      k,
		bm:        b,
		docNames:  dNames,
		plainDocs: make(chan plainDoc),
		client:    client,
	}
}

func (c *clientDaemon) search(query string) {
	c.requestSearch(query)

}

func Index(root string, db *bolt.DB) (*bloomMaster, *KeyManager, map[uint32]string) {

	i := NewIndexer()
	//	go func(i *indexer) {
	//		c.docNames = <-i.nameChannel
	//	}(i)
	return i.Index(root, db, *numWorkers)

}

func (c *clientDaemon) requestSearch(query string) {
	//TODO encrypt keyword before query
	kQuery := &pb.KeywordQuery{c.encryptQuery(query)}
	e, err := c.client.KeywordSearch(context.Background(), kQuery)
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
	//TODO later on this will actually decrypt
	buf := bytes.NewBuffer(eDoc.EncryptedId)
	val, err := binary.ReadVarint(buf)
	if err != nil {

	}
	return &pb.DocInfo{uint32(val)}

}

func (c *clientDaemon) decryptDoc(eDoc *pb.CipherDoc) []byte {
	plainTextBytes, err := c.docKey.Decrypt(eDoc.EncryptedDoc)
	//TODO store doc names
	if err != nil {
		//TODO handle error
	}
	return plainTextBytes
}
