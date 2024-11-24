package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-kad-dht"
)

type OfflineMessage struct {
	Sender    string
	Message   string
	Timestamp time.Time
}

// Generate a unique hash for DHT keys
func generateMessageHash(sender, content string, timestamp time.Time) string {
	data := sender + content + timestamp.String()
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func storeOfflineMessage(ctx context.Context, dhtNode *dht.IpfsDHT, recipientID string, message OfflineMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error serializing message:", err)
		return
	}
	key := generateMessageHash(recipientID, message.Message, message.Timestamp)
	err = dhtNode.PutValue(ctx, key, messageBytes)
	if err != nil {
		fmt.Println("Error storing message in DHT:", err)
		return
	}
	fmt.Printf("Stored message for %s: %s\n", recipientID, message.Message)
}
