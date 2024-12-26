# Raft Consensus Implementation in Go

A robust implementation of the Raft distributed consensus algorithm, providing a reliable way to manage distributed state machines.

## Features

- **Leader election with randomized timeouts**
- **Heartbeat mechanism**
- **RPC communication between nodes**
- **Thread-safe consensus module**
- **Fault tolerance support**

## Core Components

- **ConsensusModule**: The heart of the Raft implementation, managing state transitions and log replication.
- **Server**: Network layer handling RPC communications and server lifecycle.
- **RPCProxy**: Manages RPC interactions between nodes, including request votes and append entries.

## TODO

Features to be implemented:

- [ ] **Commands and log replication**: Implement client commands and ensure log entries are replicated across the cluster.
- [ ] **Persistence and optimizations**: Ensure state is persisted across restarts and optimize performance.
- [ ] **Key/Value database**: Build a simple key/value store on top of the Raft consensus module.
- [ ] **Exactly-once delivery**: Ensure that client commands are executed exactly once, even in the presence of retries.
- [ ] **Log compaction**: Implement snapshotting to manage log size.
- [ ] **Cluster membership changes**: Support for adding and removing nodes dynamically.
- [ ] **Advanced fault tolerance**: Improve handling of network partitions and node failures.
