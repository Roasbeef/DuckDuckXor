package main

import (
	"flag"
	"runtime"
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

	//i := NewCorpusReader("..")
	//i.Start()
	//for p := range i.DocOut {
	//fmt.Println("got doc", p.Name())
	//}
}
