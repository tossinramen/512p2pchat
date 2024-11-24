package main

import (
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var protocolID = protocol.ID("/p2p-chat/1.0.0")

func main() {
	// Create a new libp2p host
	host, err := libp2p.New()
	if err != nil {
		fmt.Println("Error starting libp2p host:", err)
		os.Exit(1)
	}
	defer host.Close()

	// Print the Peer ID and listening addresses
	fmt.Println("Bootstrap Peer ID:", host.ID())
	fmt.Println("Listening on:")
	for _, addr := range host.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, host.ID())
	}

	// Set a handler for incoming streams
	host.SetStreamHandler(protocolID, func(stream network.Stream) {
		buf := make([]byte, 1024)
		n, err := stream.Read(buf)
		if err != nil {
			fmt.Printf("Error reading from stream: %v\n", err)
			return
		}
		fmt.Printf("\nMessage from %s: %s\n", stream.Conn().RemotePeer(), string(buf[:n]))
	})

	// Keep the program running to allow connections
	fmt.Println("\nBootstrap peer is now running. Use the above addresses to connect.")
	select {} // Block forever
}
