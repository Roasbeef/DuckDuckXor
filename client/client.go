package main

import (
	"flag"
	"fmt"
)

var (
	keySize           = flag.Int("key_size", 32, "Key size to be used for PRF's and AES")
	documentDirectory = flag.String("doc_dir", ".", "Directory where documents to be indexed live")
)

func init() {
}

func main() {
	fmt.Println("wazzah")
}
