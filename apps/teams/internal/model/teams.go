package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cockroachdb/pebble"
)

type TeamMember struct {
	UserID  string `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
}

type Team struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Members   []TeamMember `json:"members"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type TeamResult struct {
	Team  *Team
	Found bool
}

type Teams struct {
	db *pebble.DB
}

func NewTeams(db *pebble.DB) *Teams {
	return &Teams{db: db}
}

func (t *Teams) Set(team *Team) error {
	if team.CreatedAt.IsZero() {
		team.CreatedAt = time.Now()
	}
	team.UpdatedAt = time.Now()

	value, err := json.Marshal(team)
	if err != nil {
		return fmt.Errorf("failed to marshal team: %w", err)
	}
	return t.db.Set([]byte(team.ID), value, pebble.Sync)
}

func (t *Teams) Get(id string) (*TeamResult, error) {
	value, closer, err := t.db.Get([]byte(id))
	if err != nil {
		if err == pebble.ErrNotFound {
			return &TeamResult{Found: false}, nil
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	defer closer.Close()

	var team Team
	if err := json.Unmarshal(value, &team); err != nil {
		return nil, fmt.Errorf("failed to unmarshal team: %w", err)
	}
	return &TeamResult{Team: &team, Found: true}, nil
}

func (t *Teams) Delete(id string) error {
	return t.db.Delete([]byte(id), pebble.Sync)
}

func (t *Teams) List(pageSize int32, pageToken string) ([]*Team, string, error) {
	var teams []*Team
	var startKey []byte
	if pageToken != "" {
		startKey = []byte(pageToken)
	}

	iter, _ := t.db.NewIter(&pebble.IterOptions{
		LowerBound: startKey,
	})
	defer iter.Close()

	count := int32(0)
	var nextPageToken string

	for iter.First(); iter.Valid() && count < pageSize; iter.Next() {
		team := &Team{}
		if err := json.Unmarshal(iter.Value(), team); err != nil {
			return nil, "", fmt.Errorf("failed to unmarshal team: %v", err)
		}
		teams = append(teams, team)
		count++
	}

	if iter.Valid() {
		iter.Next()
		if iter.Valid() {
			nextPageToken = string(iter.Key())
		}
	}

	return teams, nextPageToken, nil
}

func (t *Teams) AddMember(teamID string, member TeamMember) error {
	result, err := t.Get(teamID)
	if err != nil {
		return err
	}
	if !result.Found {
		return fmt.Errorf("team not found")
	}

	team := result.Team

	// Check if member already exists
	for _, m := range team.Members {
		if m.UserID == member.UserID {
			return fmt.Errorf("user already a member of the team")
		}
	}

	team.Members = append(team.Members, member)
	return t.Set(team)
}

func (t *Teams) RemoveMember(teamID string, userID string) error {
	result, err := t.Get(teamID)
	if err != nil {
		return err
	}
	if !result.Found {
		return fmt.Errorf("team not found")
	}

	team := result.Team
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
	return t.Set(team)
}

func (t *Teams) UpdateMemberRole(teamID string, userID string, isAdmin bool) error {
	result, err := t.Get(teamID)
	if err != nil {
		return err
	}
	if !result.Found {
		return fmt.Errorf("team not found")
	}

	team := result.Team
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

	return t.Set(team)
}
