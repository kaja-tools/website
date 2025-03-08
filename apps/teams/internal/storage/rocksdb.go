package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/linxGnu/grocksdb"
)

type RocksDB struct {
	db *grocksdb.DB
}

func NewRocksDB(dataDir string) (*RocksDB, error) {
	bbto := grocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(grocksdb.NewLRUCache(3 << 30)) // 3GB cache

	opts := grocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)

	path := filepath.Join(dataDir, "teams.db")
	db, err := grocksdb.OpenDb(opts, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open RocksDB: %v", err)
	}

	return &RocksDB{db: db}, nil
}

func (r *RocksDB) Close() error {
	r.db.Close()
	return nil
}

func (r *RocksDB) CreateTeam(ctx context.Context, team *Team) error {
	if team.ID == "" {
		team.ID = uuid.New().String()
	}
	team.CreatedAt = time.Now()
	team.UpdatedAt = team.CreatedAt

	data, err := json.Marshal(team)
	if err != nil {
		return fmt.Errorf("failed to marshal team: %v", err)
	}

	err = r.db.Put(grocksdb.NewDefaultWriteOptions(), []byte(team.ID), data)
	if err != nil {
		return fmt.Errorf("failed to store team: %v", err)
	}

	return nil
}

func (r *RocksDB) GetTeam(ctx context.Context, id string) (*Team, error) {
	data, err := r.db.Get(grocksdb.NewDefaultReadOptions(), []byte(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %v", err)
	}
	defer data.Free()

	if !data.Exists() {
		return nil, fmt.Errorf("team not found")
	}

	var team Team
	if err := json.Unmarshal(data.Data(), &team); err != nil {
		return nil, fmt.Errorf("failed to unmarshal team: %v", err)
	}

	return &team, nil
}

func (r *RocksDB) DeleteTeam(ctx context.Context, id string) error {
	err := r.db.Delete(grocksdb.NewDefaultWriteOptions(), []byte(id))
	if err != nil {
		return fmt.Errorf("failed to delete team: %v", err)
	}
	return nil
}

func (r *RocksDB) ListTeams(ctx context.Context, pageSize int32, pageToken string) ([]*Team, string, error) {
	opts := grocksdb.NewDefaultReadOptions()
	iter := r.db.NewIterator(opts)
	defer iter.Close()

	if pageToken != "" {
		iter.Seek([]byte(pageToken))
	} else {
		iter.SeekToFirst()
	}

	var teams []*Team
	var nextPageToken string
	count := int32(0)

	for ; iter.Valid() && count < pageSize; iter.Next() {
		var team Team
		if err := json.Unmarshal(iter.Value().Data(), &team); err != nil {
			return nil, "", fmt.Errorf("failed to unmarshal team: %v", err)
		}
		teams = append(teams, &team)
		count++
	}

	if iter.Valid() {
		nextPageToken = string(iter.Key().Data())
	}

	return teams, nextPageToken, nil
}

func (r *RocksDB) AddTeamMember(ctx context.Context, teamID string, member TeamMember) error {
	team, err := r.GetTeam(ctx, teamID)
	if err != nil {
		return err
	}

	// Check if member already exists
	for _, m := range team.Members {
		if m.UserID == member.UserID {
			return fmt.Errorf("user already a member of the team")
		}
	}

	team.Members = append(team.Members, member)
	team.UpdatedAt = time.Now()

	data, err := json.Marshal(team)
	if err != nil {
		return fmt.Errorf("failed to marshal team: %v", err)
	}

	err = r.db.Put(grocksdb.NewDefaultWriteOptions(), []byte(team.ID), data)
	if err != nil {
		return fmt.Errorf("failed to update team: %v", err)
	}

	return nil
}

func (r *RocksDB) RemoveTeamMember(ctx context.Context, teamID string, userID string) error {
	team, err := r.GetTeam(ctx, teamID)
	if err != nil {
		return err
	}

	found := false
	newMembers := make([]TeamMember, 0, len(team.Members)-1)
	for _, m := range team.Members {
		if m.UserID != userID {
			newMembers = append(newMembers, m)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("user not found in team")
	}

	team.Members = newMembers
	team.UpdatedAt = time.Now()

	data, err := json.Marshal(team)
	if err != nil {
		return fmt.Errorf("failed to marshal team: %v", err)
	}

	err = r.db.Put(grocksdb.NewDefaultWriteOptions(), []byte(team.ID), data)
	if err != nil {
		return fmt.Errorf("failed to update team: %v", err)
	}

	return nil
}

func (r *RocksDB) UpdateTeamMemberRole(ctx context.Context, teamID string, userID string, isAdmin bool) error {
	team, err := r.GetTeam(ctx, teamID)
	if err != nil {
		return err
	}

	found := false
	for i := range team.Members {
		if team.Members[i].UserID == userID {
			team.Members[i].IsAdmin = isAdmin
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("user not found in team")
	}

	team.UpdatedAt = time.Now()

	data, err := json.Marshal(team)
	if err != nil {
		return fmt.Errorf("failed to marshal team: %v", err)
	}

	err = r.db.Put(grocksdb.NewDefaultWriteOptions(), []byte(team.ID), data)
	if err != nil {
		return fmt.Errorf("failed to update team: %v", err)
	}

	return nil
}
