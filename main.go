package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
)
var consoleMu sync.Mutex

var (
	name       string
	rendezvous = "p2p-chat-room"
	protocolID = protocol.ID("/p2p-chat/1.0.0")
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: p2p-chat [bootstrap-address]")
		os.Exit(1)
	}
	bootstrapAddr := os.Args[1]

	ctx, cancel := context.WithCancel(context.Background())

	host, err := libp2p.New()
	if err != nil {
		fmt.Println("Error starting libp2p node:", err)
		os.Exit(1)
	}
	defer host.Close()

	host.SetStreamHandler(protocolID, func(stream network.Stream) {
		buf := make([]byte, 1024)
		n, err := stream.Read(buf)
		if err != nil {
			fmt.Printf("Error reading from stream: %v\n", err)
			return
		}
		message := string(buf[:n])

		consoleMu.Lock()
		fmt.Printf("\nMessage from %s: %s\n%s: ", stream.Conn().RemotePeer(), message, name)
		consoleMu.Unlock()
	})

	fmt.Print("Enter your name: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		name = strings.TrimSpace(scanner.Text())
	}
	if scanner.Err() != nil || len(name) < 2 || len(name) > 30 {
		fmt.Println("Invalid name. Must be between 2 and 30 characters.")
		return
	}

	fmt.Println("Your Peer ID:", host.ID())
	fmt.Println("Listening on:", host.Addrs())

	dhtNode := initDHT(ctx, host)
	if dhtNode == nil {
		fmt.Println("Failed to initialize DHT.")
		return
	}

	connectToBootstrapPeer(ctx, host, bootstrapAddr)

	go handleUserInput(ctx, host, scanner)


	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	<-exit


	cancel()

	fmt.Println("\nExiting chat...")
}


func connectToBootstrapPeer(ctx context.Context, host host.Host, bootstrapAddr string) {
	fmt.Println("Attempting to connect to bootstrap address:", bootstrapAddr)

	maddr, err := multiaddr.NewMultiaddr(bootstrapAddr)
	if err != nil {
		fmt.Printf("Invalid multiaddress format: %v\n", err)
		return
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		fmt.Printf("Error extracting peer info: %v\n", err)
		return
	}

	err = host.Connect(ctx, *peerInfo)
	if err != nil {
		fmt.Printf("Failed to connect to bootstrap peer at %s: %v\n", bootstrapAddr, err)
		return
	}

	fmt.Printf("Successfully connected to bootstrap peer: %s\n", peerInfo.ID)
}

func initDHT(ctx context.Context, host host.Host) *dht.IpfsDHT {
	dhtNode, err := dht.New(ctx, host)
	if err != nil {
		fmt.Println("Error initializing DHT:", err)
		return nil
	}
	if err := dhtNode.Bootstrap(ctx); err != nil {
		fmt.Println("Error bootstrapping DHT:", err)
		return nil
	}
	fmt.Println("DHT initialized and bootstrapped.")
	return dhtNode
}

func broadcastMessage(ctx context.Context, host host.Host, message string) error {
	for _, peer := range host.Network().Peers() {
		stream, err := host.NewStream(ctx, peer, protocolID)
		if err != nil {
			fmt.Printf("Error creating stream for peer %s: %v\n", peer, err)
			continue
		}
		defer stream.Close()

		_, err = stream.Write([]byte(message))
		if err != nil {
			fmt.Printf("Error writing to peer %s: %v\n", peer, err)
			continue
		}
	}
	
	return nil
}

func handleUserInput(ctx context.Context, host host.Host, scanner *bufio.Scanner) {
	for {
		select {
		case <-ctx.Done():
			
			return
		default:
			
			consoleMu.Lock()
			fmt.Printf("%s: ", name)
			consoleMu.Unlock()

			if scanner.Scan() {
				message := strings.TrimSpace(scanner.Text())
				if message == "" {
					continue 
				}

				
				formattedMessage := fmt.Sprintf("%s: %s", name, message)
				if err := broadcastMessage(ctx, host, formattedMessage); err != nil {
					consoleMu.Lock()
					fmt.Printf("Error sending message: %v\n", err)
					consoleMu.Unlock()
				}
			} else {
				return
			}
		}
	}
}
