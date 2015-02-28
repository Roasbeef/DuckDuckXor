package main

import (
	"flag"
	"runtime"
)

var (
	keySize           = flag.Int("key_size", 32, "Key size to be used for PRF's and AES")
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
