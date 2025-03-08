package storage

import (
	"context"
	"time"
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

type Storage interface {
	// Team operations
	CreateTeam(ctx context.Context, team *Team) error
	GetTeam(ctx context.Context, id string) (*Team, error)
	DeleteTeam(ctx context.Context, id string) error
	ListTeams(ctx context.Context, pageSize int32, pageToken string) ([]*Team, string, error)

	// Team member operations
	AddTeamMember(ctx context.Context, teamID string, member TeamMember) error
	RemoveTeamMember(ctx context.Context, teamID string, userID string) error
	UpdateTeamMemberRole(ctx context.Context, teamID string, userID string, isAdmin bool) error

	// Close the storage
	Close() error
}

// For pagination
type ListTeamsPage struct {
	Teams    []*Team
	NextPage string
}
