package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Create a Raft cluster with 3 servers
	numServers := 3
	servers := make([]*Server, numServers)

	for i := 0; i < numServers; i++ {
		peerIds := make([]int, 0)
		for j := 0; j < numServers; j++ {
			if j != i {
				peerIds = append(peerIds, j)
			}
		}
		servers[i] = NewServer(i, peerIds)
		servers[i].Serve()
	}

	// Connect all servers to each other
	for i := 0; i < numServers; i++ {
		for j := 0; j < numServers; j++ {
			if i != j {
				if err := servers[i].ConnectToPeer(j, servers[j].GetListenAddr()); err != nil {
					log.Fatalf("Error connecting server %d to server %d: %v", i, j, err)
				}
			}
		}
	}

	// Wait for a leader to be elected
	time.Sleep(5 * time.Second)

	// Check and log the current leader
	leaderId := -1
	leaderTerm := -1
	for i := 0; i < numServers; i++ {
		if servers[i].cm.state == Leader {
			leaderId = i
			leaderTerm = servers[i].cm.currentTerm
			break
		}
	}

	if leaderId == -1 {
		log.Fatalf("No leader elected")
	} else {
		log.Printf("Leader elected: Server %d in term %d", leaderId, leaderTerm)
	}

	// Simulate leader failure
	log.Printf("Simulating leader failure: Shutting down server %d", leaderId)
	servers[leaderId].Shutdown()

	// Wait for a new leader to be elected
	time.Sleep(5 * time.Second)

	// Check and log the new leader
	newLeaderId := -1
	newLeaderTerm := -1
	for i := 0; i < numServers; i++ {
		if i != leaderId && servers[i].cm.state == Leader {
			newLeaderId = i
			newLeaderTerm = servers[i].cm.currentTerm
			break
		}
	}

	if newLeaderId == -1 {
		log.Fatalf("No new leader elected after failure")
	} else {
		log.Printf("New leader elected: Server %d in term %d", newLeaderId, newLeaderTerm)
	}

	// Set up signal handling to gracefully shut down the servers
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	log.Println("Shutting down servers...")

	for _, server := range servers {
		if server != nil {
			server.Shutdown()
		}
	}

	log.Println("Servers shut down.")
}
