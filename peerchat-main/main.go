package main

import (
	"flag"
	"os"
	"time"

	"github.com/JustMangler/peerchat/src"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetOutput(os.Stdout)
}

func main() {

	// Parse command flags to get username
	username := flag.String("username", "guest", "Username to join the chatroom with")
	flag.Parse()

	// Initialize a new Node
	node := src.InitializeNode()
	logrus.Infoln("Completed P2P Setup")

	// Connect to peers using the specified discovery method
	node.AnnounceServiceCID()
	logrus.Infoln("Connected to Service Peers")

	// Join the chat room
	chatApp, _ := src.JoinRoom(node, *username, "lobby")
	logrus.Infof("Joined the '%s' chatroom as '%s'", chatApp.RoomName, chatApp.Username)

	// Wait for network setup to complete
	time.Sleep(5 * time.Second)

	// Create and start the Chat UI
	ui := src.NewUI(chatApp)
	ui.Run()
}