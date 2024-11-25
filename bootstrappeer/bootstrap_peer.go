package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var protocolID = protocol.ID("/p2p-chat/1.0.0")
var rendezvous = "p2p-chat-room"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	
	host, err := libp2p.New()
	if err != nil {
		fmt.Println("Error starting libp2p host:", err)
		os.Exit(1)
	}
	defer host.Close()

	
	fmt.Println("Bootstrap Peer ID:", host.ID())
	fmt.Println("Listening on:")
	for _, addr := range host.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, host.ID())
	}

	
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		fmt.Println("Error creating GossipSub:", err)
		os.Exit(1)
	}

	
	topic, err := ps.Join(rendezvous)
	if err != nil {
		fmt.Println("Error joining topic:", err)
		os.Exit(1)
	}
	defer topic.Close()

	
	sub, err := topic.Subscribe()
	if err != nil {
		fmt.Println("Error subscribing to topic:", err)
		os.Exit(1)
	}

	
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				fmt.Printf("Error reading message: %v\n", err)
				return
			}
			
			fmt.Printf("\nMessage from %s: %s\n", msg.ReceivedFrom.String(), string(msg.Data))
		}
	}()

	
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	<-exit

	fmt.Println("\nExiting bootstrap peer...")
}
