# **P2P Chat Application**

This is a decentralized peer-to-peer (P2P) chat application built using Go and `libp2p`. Follow these instructions to set up and run the application.

```
# **Prerequisites**

1. Install Go:  
   Download and install Go from https://go.dev/dl/.  
   Ensure `go` is added to your system's PATH. You can verify by running:
   go version

2. Clone the repository:  
   Clone this repository or copy the project files to your local machine.

---

# **Setup and Build**

1. Initialize dependencies:  
   Open a terminal in the project directory and run:
   go mod tidy
   This will download and install all required dependencies.

2. Build the executables:  
   Run the following commands to build the `bootstrap_peer` and `p2p_chat` executables:
   go build -o bootstrap_peer bootstrap_peer.go
   go build -o p2p-chat main.go

---

# **Running the Application**

## Step 1: Start the Bootstrap Peer
1. Start the bootstrap peer:
   ./bootstrap_peer

2. Copy the peer address printed in the terminal. It will look something like this:
   /ip4/127.0.0.1/tcp/54321/p2p/12D3KooW...

## Step 2: Start Chat Peers
1. Open a new terminal for each peer that wants to join the chat.

2. Start a peer using the following command:
   ./p2p_chat [bootstrap_address]
   Replace `[bootstrap_address]` with the address printed by the bootstrap peer in Step 1.

3. Enter your name when prompted:
   Example: Enter your name: John

4. Start chatting! Messages sent by one peer will appear in the terminals of all connected peers.

---

# **Troubleshooting**

1. Check if a Port is Open:
   If the application fails to connect, verify that the port used by the bootstrap peer or chat peer is open:
   netstat -ano | findstr [port]
   Replace `[port]` with the port number from the peer's address (e.g., 54321).

2. Resolve Dependency Issues:
   If you encounter errors related to dependencies, re-run:
   go mod tidy




