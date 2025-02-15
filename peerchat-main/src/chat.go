package src

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/sirupsen/logrus"
)

// ChatRoom defines a PubSub-based chat room.
type ChatRoom struct {
	NodeHost *Node

	IncomingMessages chan chatMsg
	OutgoingMessages chan string
	LogChannel       chan logEntry

	RoomName  string
	Username  string
	hostID    peer.ID
	cancelCtx context.CancelFunc
	roomCtx   context.Context
	topic     *pubsub.Topic
	sub       *pubsub.Subscription
}

type FileChunkMessage struct {
	FileName string
	Chunk    []byte
}

// chatMsg represents a message within the chat.
type chatMsg struct {
	Text        string `json:"text"`
	SenderID    string `json:"sender_id"`
	SenderName  string `json:"sender_name"`
	MsgType     string `json:"msg_type"` // "text" or "file"
	FileName    string `json:"file_name,omitempty"`
	ChunkIndex  int    `json:"chunk_index,omitempty"`
	TotalChunks int    `json:"total_chunks,omitempty"`
	ChunkData   []byte `json:"chunk_data,omitempty"`
}

// logEntry is used for internal logging of chat events.
type logEntry struct {
	Prefix string
	Msg    string
}

// JoinRoom initializes and returns a ChatRoom instance.
func JoinRoom(node *Node, username, room string) (*ChatRoom, error) {

	if username == "" {
		username = "guest"
	}
	if room == "" {
		room = "lobby"
	}

	// Set up the PubSub topic for the chat room
	topic, err := node.PubSub.Join(fmt.Sprintf("chatroom-%s", room))
	if err != nil {
		return nil, err
	}

	// Subscribe to the PubSub topic
	subscription, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Instantiate the ChatRoom
	chat := &ChatRoom{
		NodeHost:         node,
		IncomingMessages: make(chan chatMsg),
		OutgoingMessages: make(chan string),
		LogChannel:       make(chan logEntry),
		RoomName:         room,
		Username:         username,
		hostID:           node.Host.ID(),
		roomCtx:          ctx,
		cancelCtx:        cancel,
		topic:            topic,
		sub:              subscription,
	}

	// Start the subscription and publishing loops
	go chat.listenForMessages()
	go chat.publishMessages()

	return chat, nil
}

func (c *ChatRoom) SendFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("unable to open file: %w", err)
	}
	defer file.Close()

	const maxFileSize = 100 * 1024 // 100 KB

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Size() > maxFileSize {
		return fmt.Errorf("file size exceeds the maximum allowed size of %d bytes", maxFileSize)
	}

	fileName := filepath.Base(filePath)
	const chunkSize = 4096 // 4KB
	buf := make([]byte, chunkSize)
	var chunkIndex int
	var totalChunks int

	// Get the total size of the file to calculate total chunks
	totalChunks = int(fileInfo.Size()/chunkSize) + 1
	c.LogChannel <- logEntry{Prefix: "info", Msg: fmt.Sprintf("Sending file %s in %d chunks", fileName, totalChunks)}
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading file: %w", err)
		}
		if n == 0 {
			break
		}

		message := chatMsg{
			SenderID:    c.hostID.Pretty(),
			SenderName:  c.Username,
			MsgType:     "file",
			FileName:    fileName,
			ChunkIndex:  chunkIndex,
			TotalChunks: totalChunks,
			ChunkData:   buf[:n],
		}

		data, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("error marshaling file chunk: %w", err)
		}
		if err := c.topic.Publish(c.roomCtx, data); err != nil {
			return fmt.Errorf("error publishing file chunk: %w", err)
		}

		chunkIndex++
	}
	return nil
}

// listenForMessages handles incoming messages from the PubSub topic.
func (c *ChatRoom) listenForMessages() {
	// Map to store file chunks received
	fileChunks := make(map[string][][]byte)

	for {
		select {
		case <-c.roomCtx.Done():
			return
		default:
			msg, err := c.sub.Next(c.roomCtx)
			if err != nil {
				close(c.IncomingMessages)
				c.LogChannel <- logEntry{Prefix: "error", Msg: "Subscription closed unexpectedly"}
				return
			}

			if msg.ReceivedFrom == c.hostID {
				continue
			}

			var parsedMsg chatMsg
			if err := json.Unmarshal(msg.Data, &parsedMsg); err != nil {
				c.LogChannel <- logEntry{Prefix: "error", Msg: "Failed to parse incoming message"}
				continue
			}

			if parsedMsg.MsgType == "file" {

				// Handle file chunk
				key := parsedMsg.FileName + parsedMsg.SenderID
				if _, exists := fileChunks[key]; !exists {
					fileChunks[key] = make([][]byte, parsedMsg.TotalChunks)
				}
				fileChunks[key][parsedMsg.ChunkIndex] = parsedMsg.ChunkData

				// Check if all chunks are received
				receivedAll := true
				for _, chunk := range fileChunks[key] {
					if chunk == nil {
						receivedAll = false
						break
					}
				}

				if receivedAll {
					// Assemble the file
					go assembleAndSaveFile(parsedMsg.FileName, fileChunks[key])
					delete(fileChunks, key)
				}
			} else {
				c.IncomingMessages <- parsedMsg
			}

		}
	}
}

func assembleAndSaveFile(fileName string, chunks [][]byte) {
	filePath := filepath.Join(os.Getenv("HOME"), "Desktop", fileName)
	file, err := os.Create(filePath)
	if err != nil {
		logrus.WithError(err).Error("Failed to create file")
		return
	}
	defer file.Close()

	for _, chunk := range chunks {
		if _, err := file.Write(chunk); err != nil {
			logrus.WithError(err).Error("Failed to write file chunk")
			return
		}
	}

	// Optionally, open the file
	if err := openFile(filePath); err != nil {
		logrus.WithError(err).Error("Failed to open file")
	}

}

func openFile(filePath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", filePath)
	case "windows":
		cmd = exec.Command("explorer", filePath)
	default: // Linux and others
		cmd = exec.Command("xdg-open", filePath)
	}
	return cmd.Start()
}

// publishMessages continuously publishes outgoing messages to the topic.
func (c *ChatRoom) publishMessages() {
	for {
		select {
		case <-c.roomCtx.Done():
			return
		case msg := <-c.OutgoingMessages:
			message := chatMsg{
				Text:       msg,
				SenderID:   c.hostID.Pretty(),
				SenderName: c.Username,
			}

			data, err := json.Marshal(message)
			if err != nil {
				c.LogChannel <- logEntry{Prefix: "error", Msg: "Failed to serialize message"}
				continue
			}

			if err := c.topic.Publish(c.roomCtx, data); err != nil {
				c.LogChannel <- logEntry{Prefix: "error", Msg: "Failed to publish message"}
				continue
			}
		}
	}
}

// GetPeers retrieves a list of peers currently in the chat room.
func (c *ChatRoom) GetPeers() []peer.ID {
	return c.topic.ListPeers()
}

// Leave gracefully shuts down the chat room by closing resources.
func (c *ChatRoom) Leave() {
	defer c.cancelCtx()

	c.sub.Cancel()
	c.topic.Close()
}

// UpdateUsername allows the user to change their username.
func (c *ChatRoom) UpdateUsername(newName string) {
	c.Username = newName
}