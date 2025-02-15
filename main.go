package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"database/sql"
	_ "modernc.org/sqlite"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	tls "github.com/libp2p/go-libp2p-tls"
	yamux "github.com/libp2p/go-libp2p-yamux"
	tcp "github.com/libp2p/go-tcp-transport"
	"github.com/multiformats/go-multiaddr"
)

var consoleMu sync.Mutex

var (
	name       string
	rendezvous = "p2p-chat-room"
	protocolID = protocol.ID("/p2p-chat/1.0.0")
)

var db *sql.DB

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) < 2 {
		fmt.Println("Usage: p2p-chat [bootstrap-address]")
		os.Exit(1)
	}
	bootstrapAddr := os.Args[1]

	if err := initSQLite(); err != nil {
		fmt.Println("Failed to initialize SQLite database:", err)
		return
	}
	defer func() {
		if db != nil {
			fmt.Println("Closing SQLite database...")
			if err := db.Close(); err != nil {
				fmt.Printf("Error closing SQLite database: %v\n", err)
			} else {
				fmt.Println("SQLite database closed successfully.")
			}
		}
	}()

	host, dhtNode := setupHostAndDHT(ctx)
	defer host.Close()

	fmt.Println("Your Peer ID:", host.ID())
	fmt.Println("Listening on:", host.Addrs())

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

func setupHostAndDHT(ctx context.Context) (host.Host, *dht.IpfsDHT) {
	prvkey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		fmt.Println("Error generating key pair:", err)
		os.Exit(1)
	}

	tlstransport, err := tls.New(prvkey)
	if err != nil {
		fmt.Println("Error setting up TLS transport:", err)
		os.Exit(1)
	}

	muxer := libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport)
	transport := libp2p.Transport(tcp.NewTCPTransport)
	conn := libp2p.ConnectionManager(connmgr.NewConnManager(100, 400, time.Minute))

	var dhtNode *dht.IpfsDHT
	routing := libp2p.Routing(func(h host.Host) (peer.Routing, error) {
		var err error
		dhtNode, err = dht.New(ctx, h, dht.Mode(dht.ModeServer))
		if err != nil {
			return nil, err
		}
		if err = dhtNode.Bootstrap(ctx); err != nil {
			return nil, err
		}
		return dhtNode, nil
	})

	opts := libp2p.ChainOptions(
		libp2p.Identity(prvkey),
		libp2p.Security(tls.ID, tlstransport),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.EnableNATService(),
		libp2p.EnableAutoRelay(),
		muxer,
		transport,
		conn,
		routing,
	)

	host, err := libp2p.New(opts)
	if err != nil {
		fmt.Println("Error creating libp2p host:", err)
		os.Exit(1)
	}

	return host, dhtNode
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

func initSQLite() error {
	fmt.Println("Initializing SQLite database...")
	var err error
	db, err = sql.Open("sqlite", "p2pchat.db")
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		peer_id TEXT,
		message TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	fmt.Println("SQLite database initialized successfully.")
	return nil
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

		message := string(msg.Data)
		fmt.Printf("\rReceived message: %s\n%s: ", message, name)

		if err := storeMessage(msg.ReceivedFrom.String(), message); err != nil {
			fmt.Printf("Error storing message: %v\n", err)
		}
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

				if message == "message_history" {
					messages, err := retrieveMessages()
					if err != nil {
						fmt.Printf("Error retrieving messages: %v\n", err)
						continue
					}
					fmt.Println("Message History:")
					for _, m := range messages {
						fmt.Println(m)
					}
					continue
				}

				formattedMessage := fmt.Sprintf("%s: %s", name, message)
				if err := topic.Publish(ctx, []byte(formattedMessage)); err != nil {
					fmt.Printf("Error sending message: %v\n", err)
				}

				if err := storeMessage("self", formattedMessage); err != nil {
					fmt.Printf("Error storing message: %v\n", err)
				}
			} else {
				return
			}
		}
	}
}

func storeMessage(peerID, message string) error {
	query := `INSERT INTO messages (peer_id, message) VALUES (?, ?);`
	_, err := db.Exec(query, peerID, message)
	if err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}
	fmt.Println("Message stored successfully.")
	return nil
}

func retrieveMessages() ([]string, error) {
	rows, err := db.Query(`SELECT timestamp, peer_id, message FROM messages ORDER BY timestamp ASC;`)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %w", err)
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var timestamp, peerID, message string
		if err := rows.Scan(&timestamp, &peerID, &message); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, fmt.Sprintf("[%s] %s: %s", timestamp, peerID, message))
	}
	return messages, nil
}