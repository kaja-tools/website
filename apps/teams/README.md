# Teams Service

This is a gRPC service for managing teams. It provides APIs for creating, retrieving, deleting, and listing teams. Teams consist of users who can be either admins or members.

## Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (protoc)
- RocksDB

### Installing Prerequisites

1. Install Go:
   ```bash
   # Visit https://golang.org/dl/
   ```

2. Install Protocol Buffers compiler:
   ```bash
   # macOS
   brew install protobuf

   # Linux
   apt-get install -y protobuf-compiler
   ```

3. Install Go protobuf plugins:
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

4. Install RocksDB:
   ```bash
   # macOS
   brew install rocksdb

   # Linux
   apt-get install -y librocksdb-dev
   ```

## Setup

1. Generate gRPC code:
   ```bash
   cd proto
   ./generate.sh
   ```

2. Build the service:
   ```bash
   go build -o teams ./cmd/main.go
   ```

3. Run the service:
   ```bash
   ./teams
   ```

The service will start on port 50052.

## Docker

To build and run with Docker:

```bash
docker build -t teams-service .
docker run -p 50052:50052 teams-service
```

## API

The service provides the following gRPC APIs:

- CreateTeam: Create a new team
- GetTeam: Get team details by ID
- DeleteTeam: Delete a team by ID
- ListTeams: List all teams with pagination
- AddTeamMember: Add a user to a team
- RemoveTeamMember: Remove a user from a team
- UpdateTeamMemberRole: Update a team member's role (admin/member)

See `proto/teams.proto` for detailed API specifications. 