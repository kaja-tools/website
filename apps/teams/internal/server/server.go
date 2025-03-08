package server

import (
	"context"
	"time"

	"github.com/wham/website/apps/teams/internal/api"
	"github.com/wham/website/apps/teams/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	api.UnimplementedTeamsServer
	storage storage.Storage
}

func New(storage storage.Storage) *Server {
	return &Server{storage: storage}
}

func (s *Server) CreateTeam(ctx context.Context, req *api.CreateTeamRequest) (*api.CreateTeamResponse, error) {
	team := &storage.Team{
		Name: req.Name,
		Members: []storage.TeamMember{
			{
				UserID:  req.AdminUserId,
				IsAdmin: true,
			},
		},
	}

	if err := s.storage.CreateTeam(ctx, team); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create team: %v", err)
	}

	return &api.CreateTeamResponse{
		Team: convertTeamToProto(team),
	}, nil
}

func (s *Server) GetTeam(ctx context.Context, req *api.GetTeamRequest) (*api.GetTeamResponse, error) {
	team, err := s.storage.GetTeam(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found: %v", err)
	}

	return &api.GetTeamResponse{
		Team: convertTeamToProto(team),
	}, nil
}

func (s *Server) DeleteTeam(ctx context.Context, req *api.DeleteTeamRequest) (*api.DeleteTeamResponse, error) {
	if err := s.storage.DeleteTeam(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete team: %v", err)
	}

	return &api.DeleteTeamResponse{
		Success: true,
	}, nil
}

func (s *Server) ListTeams(ctx context.Context, req *api.ListTeamsRequest) (*api.ListTeamsResponse, error) {
	teams, nextPageToken, err := s.storage.ListTeams(ctx, req.PageSize, req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list teams: %v", err)
	}

	protoTeams := make([]*api.Team, len(teams))
	for i, team := range teams {
		protoTeams[i] = convertTeamToProto(team)
	}

	return &api.ListTeamsResponse{
		Teams:         protoTeams,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *Server) AddTeamMember(ctx context.Context, req *api.AddTeamMemberRequest) (*api.AddTeamMemberResponse, error) {
	member := storage.TeamMember{
		UserID:  req.UserId,
		IsAdmin: req.IsAdmin,
	}

	if err := s.storage.AddTeamMember(ctx, req.TeamId, member); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add team member: %v", err)
	}

	team, err := s.storage.GetTeam(ctx, req.TeamId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated team: %v", err)
	}

	return &api.AddTeamMemberResponse{
		Team: convertTeamToProto(team),
	}, nil
}

func (s *Server) RemoveTeamMember(ctx context.Context, req *api.RemoveTeamMemberRequest) (*api.RemoveTeamMemberResponse, error) {
	if err := s.storage.RemoveTeamMember(ctx, req.TeamId, req.UserId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove team member: %v", err)
	}

	team, err := s.storage.GetTeam(ctx, req.TeamId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated team: %v", err)
	}

	return &api.RemoveTeamMemberResponse{
		Team: convertTeamToProto(team),
	}, nil
}

func (s *Server) UpdateTeamMemberRole(ctx context.Context, req *api.UpdateTeamMemberRoleRequest) (*api.UpdateTeamMemberRoleResponse, error) {
	if err := s.storage.UpdateTeamMemberRole(ctx, req.TeamId, req.UserId, req.IsAdmin); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update team member role: %v", err)
	}

	team, err := s.storage.GetTeam(ctx, req.TeamId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated team: %v", err)
	}

	return &api.UpdateTeamMemberRoleResponse{
		Team: convertTeamToProto(team),
	}, nil
}

func convertTeamToProto(team *storage.Team) *api.Team {
	members := make([]*api.TeamMember, len(team.Members))
	for i, member := range team.Members {
		members[i] = &api.TeamMember{
			UserId:  member.UserID,
			IsAdmin: member.IsAdmin,
		}
	}

	return &api.Team{
		Id:        team.ID,
		Name:      team.Name,
		Members:   members,
		CreatedAt: team.CreatedAt.Format(time.RFC3339),
		UpdatedAt: team.UpdatedAt.Format(time.RFC3339),
	}
}
