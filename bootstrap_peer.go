package bootstrappeer

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// ConnectToBootstrapPeers connects the libp2p host to the predefined bootstrap peers.
func ConnectToBootstrapPeers(ctx context.Context, host host.Host, bootstrapPeers []string) {
	for _, addr := range bootstrapPeers {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			fmt.Printf("Error parsing bootstrap address: %v\n", err)
			continue
		}
		peerinfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			fmt.Printf("Error extracting peer info: %v\n", err)
			continue
		}
		err = host.Connect(ctx, *peerinfo)
		if err != nil {
			fmt.Printf("Error connecting to bootstrap peer: %v\n", err)
		} else {
			fmt.Printf("Connected to bootstrap peer: %s\n", peerinfo.ID)
		}
	}
}