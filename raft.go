package main

import (
	"log"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

type CMState int

const (
	Follower CMState = iota
	Candidate
	Leader
	Dead
)

func (s CMState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Dead:
		return "Dead"
	default:
		panic("unreachable")
	}
}

// ConsensusModule represents the core Raft consensus module
type ConsensusModule struct {
	mu sync.Mutex

	id      int
	peerIds []int
	server  *Server

	// Persistent Raft state
	currentTerm int
	votedFor    int

	// Volatile Raft state
	state CMState

	electionResetEvent time.Time
}

func NewConsensusModule(id int, peerIds []int, server *Server) *ConsensusModule {
	cm := &ConsensusModule{
		id:                 id,
		peerIds:            peerIds,
		server:             server,
		state:              Follower,
		votedFor:           -1,
		currentTerm:        0,
		electionResetEvent: time.Now(),
	}
	go cm.runElectionTimer()
	return cm
}

func (cm *ConsensusModule) electionTimeout() time.Duration {
	// Election timeout is randomly chosen between 150-300ms
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}

func (cm *ConsensusModule) runElectionTimer() {
	timeoutDuration := cm.electionTimeout()
	cm.mu.Lock()
	termStarted := cm.currentTerm
	cm.mu.Unlock()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C

		cm.mu.Lock()
		if cm.state != Candidate && cm.state != Follower {
			log.Printf("Stopping election timer for server %d (not Candidate or Follower)\n", cm.id)
			cm.mu.Unlock()
			return
		}

		if termStarted != cm.currentTerm {
			log.Printf("Stopping election timer for server %d (term changed)\n", cm.id)
			cm.mu.Unlock()
			return
		}

		// Start election if we haven't heard from leader in timeoutDuration
		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
			log.Printf("Starting election for server %d\n", cm.id)
			// cm.startElection()
			cm.mu.Unlock()
			return
		}
		cm.mu.Unlock()
	}
}
