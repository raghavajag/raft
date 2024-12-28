package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

const DebugCM = 1

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

type AppendEntriesArgs struct {
	Term     int
	LeaderId int

	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}
type LogEntry struct {
	Command any
	Term    int
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
	state ServerState

	electionResetEvent time.Time

	log []LogEntry
}

func NewConsensusModule(id int, peerIds []int, server *Server, ready <-chan any) *ConsensusModule {
	cm := &ConsensusModule{
		id:          id,
		peerIds:     peerIds,
		server:      server,
		state:       Follower,
		votedFor:    -1,
		currentTerm: 0,
	}

	go func() {
		<-ready
		cm.mu.Lock()
		cm.electionResetEvent = time.Now()
		cm.mu.Unlock()
		cm.runElectionTimer()
	}()
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
		/*
			This condition checks if the node is still in the `Candidate` or `Follower` state.
			If the node has become a `Leader` or `Dead`, the election timer is stopped because
			a leader does not need to start an election, and a dead node should not participate in elections.
		*/
		if cm.state != Candidate && cm.state != Follower {
			cm.dlog("Stopping election timer (not Candidate or Follower)")
			cm.mu.Unlock()
			return
		}
		/*
			This condition checks if the term has changed since the election timer started. If the term has changed,
			it means that either this node has received a higher term from another node (indicating a new leader) or
			this node has started a new term itself. In either case, the current election timer should be stopped.
		*/
		if termStarted != cm.currentTerm {
			cm.dlog("Stopping election timer (term changed)")
			cm.mu.Unlock()
			return
		}

		// Start election if we haven't heard from leader in timeoutDuration
		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
			cm.dlog("Starting election")
			cm.startElection()
			cm.mu.Unlock()
			return
		}
		cm.mu.Unlock()
	}
}

func (cm *ConsensusModule) startElection() {
	cm.dlog("starting election for term %d", cm.currentTerm+1)
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
						cm.dlog("received vote from %d", peerId)
						if votesReceived*2 > len(cm.peerIds)+1 {
							// Won the election!
							cm.dlog("won election with %d votes", votesReceived)
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
	cm.dlog("becomes Follower with term=%d; log=%v", term, cm.log)
	cm.state = Follower
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()

	go cm.runElectionTimer()
}

func (cm *ConsensusModule) startLeader() {
	cm.dlog("becomes Leader; term=%d, log=%v", cm.currentTerm, cm.log)
	cm.state = Leader

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		// Send periodic heartbeats, as long as still leader.
		for {
			cm.leaderSendHeartbeats()
			<-ticker.C

			cm.mu.Lock()
			if cm.state != Leader {
				cm.mu.Unlock()
				return
			}
			cm.mu.Unlock()
		}
	}()
}

func (cm *ConsensusModule) leaderSendHeartbeats() {
	cm.mu.Lock()
	if cm.state != Leader {
		cm.mu.Unlock()
		return
	}
	savedCurrentTerm := cm.currentTerm
	cm.mu.Unlock()

	for _, peerId := range cm.peerIds {
		args := AppendEntriesArgs{
			Term:     savedCurrentTerm,
			LeaderId: cm.id,
		}
		go func(peerId int) {
			var reply AppendEntriesReply
			if err := cm.server.Call(peerId, "ConsensusModule.AppendEntries", args, &reply); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				if reply.Term > savedCurrentTerm {
					cm.becomeFollower(reply.Term)
					return
				}
			}
		}(peerId)
	}
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
		cm.dlog("... granted vote for term %d to %d", args.Term, args.CandidateId)
	} else {
		reply.VoteGranted = false
		cm.dlog("... denied vote for term %d to %d", args.Term, args.CandidateId)
	}
	reply.Term = cm.currentTerm
	return nil
}

func (cm *ConsensusModule) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.state == Dead {
		return nil
	}

	if args.Term > cm.currentTerm {
		cm.becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == cm.currentTerm {
		if cm.state != Follower {
			cm.becomeFollower(args.Term)
		}
		cm.electionResetEvent = time.Now()
		reply.Success = true
		cm.dlog("received heartbeat from %d for term %d", args.LeaderId, args.Term)
	}

	reply.Term = cm.currentTerm
	return nil
}

// dlog logs a debugging message if DebugCM > 0.
func (cm *ConsensusModule) dlog(format string, args ...any) {
	if DebugCM > 0 {
		format = fmt.Sprintf("[%d] ", cm.id) + format
		log.Printf(format, args...)
	}
}

func (cm *ConsensusModule) Report() (id int, term int, isLeader bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.id, cm.currentTerm, cm.state == Leader
}
