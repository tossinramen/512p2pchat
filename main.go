package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

var consoleMu sync.Mutex

var (
	name       string
	rendezvous = "p2p-chat-room"
	protocolID = protocol.ID("/p2p-chat/1.0.0")
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) < 2 {
		fmt.Println("Usage: p2p-chat [bootstrap-address]")
		os.Exit(1)
	}
	bootstrapAddr := os.Args[1]

	
	host, err := libp2p.New()
	if err != nil {
		fmt.Println("Error starting libp2p node:", err)
		os.Exit(1)
	}
	defer host.Close()

	fmt.Println("Your Peer ID:", host.ID())
	fmt.Println("Listening on:", host.Addrs())

	
	dhtNode := initDHT(ctx, host)
	if dhtNode == nil {
		fmt.Println("Failed to initialize DHT.")
		return
	}

	
	connectToBootstrapPeer(ctx, host, bootstrapAddr)

	
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		fmt.Println("Error creating GossipSub:", err)
		return
	}

	
	topic, err := ps.Join(rendezvous)
	if err != nil {
		fmt.Println("Error joining topic:", err)
		return
	}
	defer topic.Close()

	
	sub, err := topic.Subscribe()
	if err != nil {
		fmt.Println("Error subscribing to topic:", err)
		return
	}

	fmt.Print("Enter your name: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		name = strings.TrimSpace(scanner.Text())
	}
	if scanner.Err() != nil || len(name) < 2 || len(name) > 30 {
		fmt.Println("Invalid name. Must be between 2 and 30 characters.")
		return
	}


	go handleIncomingMessages(ctx, sub, host)


	go handleUserInput(ctx, topic, scanner)

	
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	<-exit

	fmt.Println("\nExiting chat...")
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


func handleIncomingMessages(ctx context.Context, sub *pubsub.Subscription, host host.Host) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			fmt.Printf("Error reading message: %v\n", err)
			return
		}

		
		if msg.ReceivedFrom == host.ID() { 
			continue
		}

		
		consoleMu.Lock()
		fmt.Printf("\n%s: %s\n%s: ", msg.ReceivedFrom.String(), string(msg.Data), name)
		consoleMu.Unlock()
	}
}




func handleUserInput(ctx context.Context, topic *pubsub.Topic, scanner *bufio.Scanner) {
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

				if strings.HasPrefix(message, "/sendimg") {
					parts := strings.SplitN(message, " ", 2)
					if len(parts) < 2 {
						fmt.Println("Usage: /sendimg [image_path]")
						continue
					}

					imagePath := parts[1]
					base64Data, err := encodeImageToBase64(imagePath)
					if err != nil {
						fmt.Printf("Failed to encode image: %v\n", err)
						continue
					}

					formattedMessage := "IMG:" + base64Data
					if err := topic.Publish(ctx, []byte(formattedMessage)); err != nil {
						fmt.Printf("Error sending image: %v\n", err)
					}
				} else {
					formattedMessage := fmt.Sprintf("%s: %s", name, message)
					if err := topic.Publish(ctx, []byte(formattedMessage)); err != nil {
						fmt.Printf("Error sending message: %v\n", err)
					}
				}
			} else {
				return
			}
		}
	}
}


func encodeImageToBase64(imagePath string) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	return base64.StdEncoding.EncodeToString(data), nil
}


func decodeBase64ToImage(encodedString, outputPath string) error {
	data, err := base64.StdEncoding.DecodeString(encodedString)
	if err != nil {
		return fmt.Errorf("failed to decode base64 string: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}
