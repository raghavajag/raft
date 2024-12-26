package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	// Command line parsing
	var servers int
	flag.IntVar(&servers, "servers", 3, "number of servers in the cluster")
	flag.Parse()

	// Create a slice to hold all servers
	serverList := make([]*Server, servers)

	// Create peer IDs for each server
	for i := 0; i < servers; i++ {
		peerIds := make([]int, 0)
		for j := 0; j < servers; j++ {
			if j != i {
				peerIds = append(peerIds, j)
			}
		}

		// Create and start each server
		server := NewServer(i, peerIds)
		serverList[i] = server
		server.Serve()

		fmt.Printf("Started server %d\n", i)
	}

	// Print all server addresses
	fmt.Println("\nServer addresses:")
	for i, server := range serverList {
		fmt.Printf("Server %d: %s\n", i, server.GetListenAddr())
	}

	// Handle Ctrl+C gracefully
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)

	// Wait for interrupt signal
	<-terminate
	fmt.Println("\nReceived interrupt signal. Shutting down...")

	// Shutdown all servers
	for i, server := range serverList {
		server.Shutdown()
		fmt.Printf("Server %d shutdown complete\n", i)
	}
}
