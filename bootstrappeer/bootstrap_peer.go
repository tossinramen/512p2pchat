package bootstrappeer

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

var protocolID = protocol.ID("/p2p-chat/1.0.0")

// ConnectToBootstrapPeers connects to provided bootstrap peers
func ConnectToBootstrapPeers(ctx context.Context, host host.Host, bootstrapPeers []string) {
	// Set the protocol handler for the bootstrap peer
	host.SetStreamHandler(protocolID, func(stream network.Stream) {
		// Read incoming messages
		buf := make([]byte, 1024)
		n, err := stream.Read(buf)
		if err != nil {
			fmt.Printf("Error reading from stream: %v\n", err)
			return
		}
		fmt.Printf("\nMessage from %s: %s\n", stream.Conn().RemotePeer(), string(buf[:n]))
	})

	// Attempt to connect to each bootstrap peer
	for _, addr := range bootstrapPeers {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			fmt.Printf("Error parsing bootstrap address: %v\n", err)
			continue
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			fmt.Printf("Error extracting peer info: %v\n", err)
			continue
		}
		err = host.Connect(ctx, *peerInfo)
		if err != nil {
			fmt.Printf("Error connecting to bootstrap peer: %v\n", err)
		} else {
			fmt.Printf("Connected to bootstrap peer: %s\n", peerInfo.ID)
		}
	}
}
