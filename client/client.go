package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"flag"
	"io"
	"runtime"

	"github.com/conformal/btcwallet/snacl"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
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

type plainDoc struct {
	plainbytes []byte
	docId      uint32
}

type clientDaemon struct {
	eDocs  chan *pb.EncryptedDocInfo
	keys   KeyManager
	docKey snacl.CryptoKey
}

func (c *clientDaemon) search(query string) {
	c.requestSearch(query)

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
		p := pb.PlainDoc{cDoc.DocId, b}
		if p.DocBytes != nil {

		}
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
