# **DHT-based P2P Chat Application in Go**  
**Authors:** Konnie Huang, Kelly Huang, William Zhou, Tahsin Rahman  

## **Overview**  
Our Peer-to-Peer (P2P) Chat Application is a decentralized communication system built in Go using the **libp2p** networking stack. It enables multiple users to interact in real-time through dynamically created chat rooms without relying on a central server. The application features a **terminal-based UI** for ease of interaction and supports **secure, decentralized communication** using **PubSub messaging and Kademlia DHT** for peer discovery.

## **Implementation Overview**  
The application consists of modular components:

1. **P2P Networking Layer:** Manages peer discovery, message propagation, and decentralized communication.
2. **Chat Room Management:** Handles subscriptions, message serialization, and room lifecycle.
3. **User Interface:** A terminal-based UI using `tview` for interactive messaging.
4. **Main Execution Logic:** Orchestrates initialization, configuration, and system execution.

## **Implementation Details**  

### **P2P Networking Layer (`p2p.go`)**  
- A **libp2p host** is initialized to enable secure communication, NAT traversal, and connection management.  
- **Kademlia DHT** is used for peer discovery, ensuring decentralized, scalable lookup of chat participants.  
- **libp2p-PubSub** is employed for broadcasting messages within chat rooms, with each chat room mapped to a unique topic.  
- **TLS encryption** is used for secure communication.

### **Chat Room Management (`chat.go`)**  
- Each chat room corresponds to a **PubSub topic**. Users subscribe to topics dynamically to exchange messages in real-time.  
- Messages are **serialized in JSON**, containing the sender's ID, name, and message text.  
- The system supports **file transfer** by breaking large files into **Base64-encoded chunks** before broadcasting them via PubSub.  
- On reception, peers reconstruct the file and store it locally.

### **User Interface (`ui.go`)**  
- Implemented using the **tview** library for a dynamic terminal-based UI.  
- Displays **chat messages**, **peer lists**, and **input commands**.  
- Users can issue commands:  
  - `/quit` - Exit the application.  
  - `/r <roomname>` - Switch chat rooms.  
  - `/u <username>` - Change username.  
  - `/send <filename>` - Send a text file or image.  
- The interface dynamically updates with messages, connected peers, and system logs.

## **Sending Text Files and Images**  
- The app **encodes files as Base64** and broadcasts them via **PubSub messaging**.  
- Files are **split into chunks** before being sent, and peers **reconstruct** them upon reception.  
- This ensures efficient, decentralized file sharing without relying on external servers.  

## **Main Application Logic (`main.go`)**  
- Initializes the **libp2p node** and **bootstraps the DHT** for peer discovery.  
- Joins a **default or user-specified chat room** and subscribes to the relevant PubSub topic.  
- Listens for incoming messages, updates the UI, and allows seamless room switching.  
- On exit, it **cleans up resources**, unsubscribes from topics, and disconnects from peers.

## **Application Flow**  
1. The user **initializes the P2P node** and connects to bootstrap peers.  
2. The user **joins a chat room**, where messages are exchanged using **PubSub**.  
3. The interface **dynamically updates** to reflect **new messages and connected peers**.  
4. Users can **switch rooms** dynamically while ensuring proper **resource cleanup**.  
5. On exit, all **connections and subscriptions** are gracefully closed.

## **Architecture Diagram**  
- **Libp2p Host:** Manages P2P communication (TCP/UDP, TLS encryption).  
- **Kademlia DHT:** Enables peer discovery and stores room metadata.  
- **Chat Rooms:** Logical **PubSub topics** for group messaging.  
- **PubSub Messaging:** Broadcasts messages to all subscribed peers.  

## **Challenges and Solutions**  

### **1. NAT Traversal**  
- Many peers operate behind **NAT firewalls**, making direct connections difficult.  
- We implemented **libp2p’s NAT traversal**, **UPnP**, and **STUN** to determine external IPs.  
- **Relay nodes** were used to forward traffic when direct connections were impossible.  

### **2. Username Uniqueness**  
- Decentralized systems lack a **central authority** to enforce unique usernames.  
- We relied on **unique peer IDs** under the hood while allowing duplicate usernames.  
- Future improvements may include **DHT-based username mapping** or **appending random identifiers** to duplicate usernames.

## **Key Learnings**  
- **Go’s concurrency model** and type system provided reliability for networking tasks.  
- We gained hands-on experience with **decentralized communication** using **DHT and PubSub**.  
- **Kademlia DHT** taught us how distributed systems **achieve scalable peer discovery**.  
- **NAT traversal** and **relay-based communication** were crucial for peer connectivity in real-world settings.  

## **Future Improvements**  
- **DHT-based username registration** for enforcing uniqueness.  
- **Enhanced file-sharing mechanisms** with better chunking strategies.  
- **Web-based UI integration** for improved accessibility.

This project provided **valuable hands-on experience** with decentralized systems, peer-to-peer networking, and Go’s powerful concurrency model.
