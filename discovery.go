package main

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

func discoverPeers(ctx context.Context, routingDiscovery *routing.RoutingDiscovery, host host.Host, name string) {
	peerChan, err := routingDiscovery.FindPeers(ctx, rendezvous)
	if err != nil {
		fmt.Printf("Error finding peers: %v\n", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context canceled, stopping peer discovery.")
			return
		case peer, ok := <-peerChan:
			if !ok {
				fmt.Println("Peer discovery channel closed.")
				return
			}

			// Skip if peer is self
			if peer.ID == host.ID() {
				continue
			}

			// Check if already connected
			if host.Network().Connectedness(peer.ID) == network.Connected {
				fmt.Printf("Already connected to peer: %s\n", peer.ID)
				continue
			}

			// Attempt to connect to the discovered peer
			fmt.Printf("Discovered peer: %s\n", peer.ID)
			err := host.Connect(ctx, peer)
			if err != nil {
				fmt.Printf("Failed to connect to peer %s: %v\n", peer.ID, err)
			} else {
				fmt.Printf("Connected to peer: %s\n", peer.ID)
				handleOutgoingMessage(ctx, host, fmt.Sprintf("%s has joined the chat.", name))
			}
		}
	}
}
