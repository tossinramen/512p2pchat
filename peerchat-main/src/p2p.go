
package src

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	tls "github.com/libp2p/go-libp2p-tls"
	yamux "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-tcp-transport"
	"github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/sirupsen/logrus"
)

const p2pServiceName = "peerchat/service"

type Node struct {
	Context   context.Context
	Host      host.Host
	DHT       *dht.IpfsDHT
	Discovery *discovery.RoutingDiscovery
	PubSub    *pubsub.PubSub
}

// InitializeNode sets up and returns a new P2P node.
func InitializeNode() *Node {
	mainCtx := context.Background()
	p2pHost, kademliaDHT := createHost(mainCtx)
	initializeDHT(mainCtx, p2pHost, kademliaDHT)
	discoveryService := discovery.NewRoutingDiscovery(kademliaDHT)
	pubSubSystem := initializePubSub(mainCtx, p2pHost, discoveryService)

	return &Node{
		Context:   mainCtx,
		Host:      p2pHost,
		DHT:       kademliaDHT,
		Discovery: discoveryService,
		PubSub:    pubSubSystem,
	}
}

// AnnounceServiceCID connects to peers providing the same CID.
func (n *Node) AnnounceServiceCID() {
	serviceCID := generateServiceCID(p2pServiceName)
	if err := n.DHT.Provide(n.Context, serviceCID, true); err != nil {
		logrus.WithError(err).Fatal("Failed to announce service CID")
	}

	time.Sleep(5 * time.Second)
	providerStream := n.DHT.FindProvidersAsync(n.Context, serviceCID, 0)
	go connectToDiscoveredPeers(n.Host, providerStream)
}

// createHost configures and returns a libp2p host and its DHT.
func createHost(ctx context.Context) (host.Host, *dht.IpfsDHT) {
	privateKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to generate private key")
	}

	listenAddr, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	tlsTransport, _ := tls.New(privateKey)

	var kadDHT *dht.IpfsDHT
	hostNode, err := libp2p.New(ctx,
		libp2p.Identity(privateKey),
		libp2p.ListenAddrs(listenAddr),
		libp2p.Security(tls.ID, tlsTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		libp2p.ConnectionManager(connmgr.NewConnManager(100, 400, time.Minute)),
		libp2p.NATPortMap(),
		libp2p.EnableAutoRelay(),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			kadDHT = initializeKademliaDHT(ctx, h)
			return kadDHT, nil
		}),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create host")
	}

	return hostNode, kadDHT
}

// initializeKademliaDHT configures and returns a Kademlia DHT.
func initializeKademliaDHT(ctx context.Context, h host.Host) *dht.IpfsDHT {
	dhtNode, _ := dht.New(ctx, h, dht.Mode(dht.ModeServer), dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...))
	return dhtNode
}

// initializeDHT bootstraps the DHT to connect to peers.
func initializeDHT(ctx context.Context, h host.Host, dhtNode *dht.IpfsDHT) {
	if err := dhtNode.Bootstrap(ctx); err != nil {
		logrus.WithError(err).Fatal("Failed to bootstrap DHT")
	}

	var wg sync.WaitGroup
	for _, bootstrapAddr := range dht.DefaultBootstrapPeers {
		peerInfo, _ := peer.AddrInfoFromP2pAddr(bootstrapAddr)
		wg.Add(1)
		go func(info peer.AddrInfo) {
			defer wg.Done()
			h.Connect(ctx, info)
		}(*peerInfo)
	}
	wg.Wait()
	logrus.Info("Bootstrapped DHT and connected to peers")
}

// initializePubSub sets up a PubSub system with discovery.
func initializePubSub(ctx context.Context, h host.Host, discoveryService *discovery.RoutingDiscovery) *pubsub.PubSub {
	pubSubSystem, err := pubsub.NewGossipSub(ctx, h, pubsub.WithDiscovery(discoveryService))
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize PubSub system")
	}
	return pubSubSystem
}

// connectToDiscoveredPeers handles connecting to peers from a channel.
func connectToDiscoveredPeers(h host.Host, peerStream <-chan peer.AddrInfo) {
	for peerInfo := range peerStream {
		if peerInfo.ID != h.ID() {
			h.Connect(context.Background(), peerInfo)
		}
	}
}

// generateServiceCID creates a CID for a given service name.
func generateServiceCID(name string) cid.Cid {
	hasher := sha256.New()
	hasher.Write([]byte(name))
	hashBytes := append([]byte{0x12, 0x20}, hasher.Sum(nil)...)
	multiHash, err := multihash.FromB58String(base58.Encode(hashBytes))
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create CID")
	}
	return cid.NewCidV1(12, multiHash)
}