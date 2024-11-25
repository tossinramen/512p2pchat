
Install Go: https://go.dev/dl/
In terminal, type in:
go mod tidy
go build
./bootstrap_peer
./p2p-chat [address]
netstat -ano | findstr [port] to check if open 