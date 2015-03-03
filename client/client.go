package main

import (
	"flag"
	"runtime"
)

var (
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
