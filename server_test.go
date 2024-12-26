package main

import (
	"testing"
	"time"
)

func TestServerCluster(t *testing.T) {
	// Create 3 servers
	servers := make([]*Server, 3)
	for i := 0; i < 3; i++ {
		peerIds := make([]int, 0)
		for j := 0; j < 3; j++ {
			if j != i {
				peerIds = append(peerIds, j)
			}
		}
		servers[i] = NewServer(i, peerIds)
		servers[i].Serve()
	}

	// Verify all servers are running
	for i, server := range servers {
		addr := server.GetListenAddr()
		if addr == nil {
			t.Errorf("Server %d: expected non-nil address", i)
		}
		t.Logf("Server %d listening on %s", i, addr)
	}

	// Let servers run for a bit
	time.Sleep(1 * time.Second)

	// Shutdown all servers
	for i, server := range servers {
		server.Shutdown()
		t.Logf("Server %d shutdown complete", i)
	}
}

func TestSingleServer(t *testing.T) {
	// Create a single server with no peers
	server := NewServer(1, []int{})

	// Start the server
	server.Serve()

	// Get and verify the address
	addr := server.GetListenAddr()
	if addr == nil {
		t.Error("expected non-nil address")
	}
	t.Logf("Server listening on %s", addr)

	// Let it run briefly
	time.Sleep(200 * time.Millisecond)

	// Shutdown
	server.Shutdown()
}

func TestPeerConnection(t *testing.T) {
	// Create 3 servers
	servers := make([]*Server, 3)
	for i := 0; i < 3; i++ {
		peerIds := make([]int, 0)
		for j := 0; j < 3; j++ {
			if j != i {
				peerIds = append(peerIds, j)
			}
		}
		servers[i] = NewServer(i, peerIds)
		servers[i].Serve()
	}

	// Connect the peers
	for i, server := range servers {
		for j, peer := range servers {
			if i != j {
				err := server.ConnectToPeer(j, peer.GetListenAddr())
				if err != nil {
					t.Fatalf("failed to connect server %d to peer %d: %v", i, j, err)
				}
			}
		}
	}

	// Verify all servers are in Follower state
	for i, server := range servers {
		if server.cm.state != Follower {
			t.Errorf("server %d: expected Follower state, got %v", i, server.cm.state)
		}
	}

	// Test disconnection
	for i, server := range servers {
		for j := 0; j < 3; j++ {
			if i != j {
				err := server.DisconnectPeer(j)
				if err != nil {
					t.Errorf("failed to disconnect server %d from peer %d: %v", i, j, err)
				}
			}
		}
	}

	// Cleanup
	for _, server := range servers {
		server.Shutdown()
	}
}

func TestConsensusModuleInitialization(t *testing.T) {
	server := NewServer(1, []int{2, 3})
	server.Serve()

	if server.cm == nil {
		t.Fatal("ConsensusModule not initialized")
	}

	if server.cm.state != Follower {
		t.Errorf("Expected initial state Follower, got %v", server.cm.state)
	}

	if server.cm.currentTerm != 0 {
		t.Errorf("Expected initial term 0, got %d", server.cm.currentTerm)
	}

	if server.cm.votedFor != -1 {
		t.Errorf("Expected initial votedFor -1, got %d", server.cm.votedFor)
	}

	server.Shutdown()
}
