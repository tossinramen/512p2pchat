package main

import (
	
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	
)


func handleOutgoingMessage(ctx context.Context, host host.Host, message string) {
	peers := host.Network().Peers()
	if len(peers) == 0 {
		fmt.Println("No peers connected. Message not sent.")
		return
	}
	for _, peerID := range peers {
		fmt.Printf("Attempting to send message to peer: %s\n", peerID)

		stream, err := host.NewStream(ctx, peerID, "/p2p-chat/1.0.0")
		if err != nil {
			fmt.Printf("Failed to create stream to peer %s: %v\n", peerID, err)
			continue
		}
		_, err = stream.Write([]byte(message + "\n"))
		if err != nil {
			fmt.Printf("Failed to send message to peer %s: %v\n", peerID, err)
		}
		stream.Close()
	}
}
