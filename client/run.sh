rm k.txt && touch k.txt
go build
./client | grep WaitGroup
