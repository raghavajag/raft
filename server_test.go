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
