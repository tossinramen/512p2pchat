package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	routingdiscovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
)

var (
	name            string
	rendezvous      = "p2p-chat-room"
	activeChatrooms = make(map[string]*Chatroom)
)

type Chatroom struct {
	Name         string
	MaxUsers     int
	CurrentUsers []string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: p2p-chat [bootstrap-address]")
		os.Exit(1)
	}
	bootstrapAddr := os.Args[1]

	ctx := context.Background()

	host, err := libp2p.New()
	if err != nil {
		fmt.Println("Error starting libp2p node:", err)
		os.Exit(1)
	}
	defer host.Close()

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
	routingDiscovery := routingdiscovery.NewRoutingDiscovery(dhtNode)

	connectToBootstrapPeer(ctx, host, bootstrapAddr)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-exit:
			return
		default:
			fmt.Print("> ")
			if scanner.Scan() {
				command := scanner.Text()
				handleUserCommand(ctx, host, routingDiscovery, command)
			}
		}
	}
}

func connectToBootstrapPeer(ctx context.Context, host host.Host, bootstrapAddr string) {
	maddr, err := multiaddr.NewMultiaddr(bootstrapAddr)
	if err != nil {
		fmt.Println("Invalid multiaddress:", err)
		os.Exit(1)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		fmt.Println("Error extracting peer info from multiaddress:", err)
		os.Exit(1)
	}

	// Connect to the bootstrap peer
	err = host.Connect(ctx, *peerInfo)
	if err != nil {
		fmt.Println("Failed to connect to bootstrap peer:", err)
		os.Exit(1)
	}
	fmt.Printf("Connected to bootstrap peer: %s\n", peerInfo.ID)
}

func handleUserCommand(ctx context.Context, host host.Host, routingDiscovery *routingdiscovery.RoutingDiscovery, command string) {
	parts := strings.Fields(command)
	if len(parts) < 2 {
		fmt.Println("Invalid command. Please try again.")
		return
	}

	switch parts[0] {
	case "create":
		if len(parts) != 4 || parts[1] != "chatroom" {
			fmt.Println("Usage: create chatroom [name] [number of participants allowed]")
			return
		}
		chatroomName := parts[2]
		maxUsers, err := strconv.Atoi(parts[3])
		if err != nil || maxUsers < 1 {
			fmt.Println("Invalid number of participants. Please enter a positive integer.")
			return
		}
		createChatroom(chatroomName, maxUsers)
	case "join":
		if len(parts) != 3 || parts[1] != "chatroom" {
			fmt.Println("Usage: join chatroom [name]")
			return
		}
		chatroomName := parts[2]
		joinChatroom(chatroomName)
	default:
		fmt.Println("Unknown command. Please try again.")
	}
}

func createChatroom(name string, maxUsers int) {
	if _, exists := activeChatrooms[name]; exists {
		fmt.Printf("Chatroom '%s' already exists.\n", name)
		return
	}
	activeChatrooms[name] = &Chatroom{Name: name, MaxUsers: maxUsers}
	fmt.Printf("Chatroom '%s' created, allowing up to %d participants.\n", name, maxUsers)
}

func joinChatroom(name string) {
	chatroom, exists := activeChatrooms[name]
	if !exists {
		fmt.Printf("Chatroom '%s' does not exist.\n", name)
		return
	}
	if len(chatroom.CurrentUsers) >= chatroom.MaxUsers {
		fmt.Printf("Chatroom '%s' is full.\n", name)
		return
	}
	chatroom.CurrentUsers = append(chatroom.CurrentUsers, name)
	fmt.Printf("Joining chatroom '%s'...\n", name)
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
