package main

import (
	"log"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

type CMState int

type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

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
			cm.startElection()
			cm.mu.Unlock()
			return
		}
		cm.mu.Unlock()
	}
}

func (cm *ConsensusModule) startElection() {
	log.Printf("Server %d starting election for term %d\n", cm.id, cm.currentTerm+1)
	cm.state = Candidate
	cm.currentTerm += 1
	savedCurrentTerm := cm.currentTerm
	cm.votedFor = cm.id
	cm.electionResetEvent = time.Now()

	votesReceived := 1

	// Send RequestVote RPCs to all peers
	for _, peerId := range cm.peerIds {
		go func(peerId int) {
			args := RequestVoteArgs{
				Term:        savedCurrentTerm,
				CandidateId: cm.id,
			}
			var reply RequestVoteReply

			if err := cm.server.Call(peerId, "ConsensusModule.RequestVote", args, &reply); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()

				if cm.state != Candidate {
					return
				}

				if reply.Term > savedCurrentTerm {
					cm.becomeFollower(reply.Term)
					return
				} else if reply.Term == savedCurrentTerm {
					if reply.VoteGranted {
						votesReceived += 1
						log.Printf("Server %d received vote from server %d\n", cm.id, peerId)
						if votesReceived*2 > len(cm.peerIds)+1 {
							// Won the election!
							log.Printf("Server %d won the election for term %d\n", cm.id, savedCurrentTerm)
							cm.startLeader()
							return
						}
					}
				}
			}
		}(peerId)
	}

	// Start another election timer, in case this election not successful
	go cm.runElectionTimer()
}

func (cm *ConsensusModule) becomeFollower(term int) {
	log.Printf("Server %d becoming follower for term %d\n", cm.id, term)
	cm.state = Follower
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()

	go cm.runElectionTimer()
}

func (cm *ConsensusModule) startLeader() {
	log.Printf("Server %d becoming leader for term %d\n", cm.id, cm.currentTerm)
	cm.state = Leader
	// Send periodic heartbeats, as long as still leader.
}

func (cm *ConsensusModule) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.state = Dead
}

func (cm *ConsensusModule) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.state == Dead {
		return nil
	}

	if args.Term > cm.currentTerm {
		cm.becomeFollower(args.Term)
	}

	if cm.currentTerm == args.Term &&
		(cm.votedFor == -1 || cm.votedFor == args.CandidateId) {
		reply.VoteGranted = true
		cm.votedFor = args.CandidateId
		cm.electionResetEvent = time.Now()
		log.Printf("Server %d granted vote to server %d for term %d\n", cm.id, args.CandidateId, args.Term)
	} else {
		reply.VoteGranted = false
		log.Printf("Server %d did not grant vote to server %d for term %d\n", cm.id, args.CandidateId, args.Term)
	}
	reply.Term = cm.currentTerm
	return nil
}
