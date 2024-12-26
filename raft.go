package main

import "sync"

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
}

func NewConsensusModule(id int, peerIds []int, server *Server) *ConsensusModule {
    cm := &ConsensusModule{
        id:          id,
        peerIds:     peerIds,
        server:      server,
        state:       Follower,
        votedFor:    -1,
        currentTerm: 0,
    }
    return cm
}
