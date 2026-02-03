package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kaja-tools/website/v2/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type teamsServer struct {
	UnimplementedTeamsServer
	model *model.Teams
}

func NewTeamsServer(model *model.Teams) TeamsServer {
	return &teamsServer{model: model}
}

func (s *teamsServer) CreateTeam(ctx context.Context, req *CreateTeamRequest) (*CreateTeamResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "team name cannot be empty")
	}

	team := &model.Team{
		ID:        uuid.New().String(),
		Name:      req.Name,
		CreatedAt: time.Now(),
	}

	if err := s.model.Set(team); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create team: %v", err)
	}

	return &CreateTeamResponse{
		Team: convertTeamToProto(team),
	}, nil
}

func (s *teamsServer) GetTeam(ctx context.Context, req *GetTeamRequest) (*GetTeamResponse, error) {
	if req.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "team ID cannot be empty")
	}

	result, err := s.model.Get(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get team: %v", err)
	}
	if !result.Found {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}

	return &GetTeamResponse{
		Team: convertTeamToProto(result.Team),
	}, nil
}

func (s *teamsServer) DeleteTeam(ctx context.Context, req *DeleteTeamRequest) (*DeleteTeamResponse, error) {
	if req.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "team ID cannot be empty")
	}

	result, err := s.model.Get(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get team: %v", err)
	}
	if !result.Found {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}

	if err := s.model.Delete(req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete team: %v", err)
	}

	return &DeleteTeamResponse{}, nil
}

func (s *teamsServer) GetAllTeams(ctx context.Context, req *GetAllTeamsRequest) (*GetAllTeamsResponse, error) {
	teams, err := s.model.GetAll()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get all teams: %v", err)
	}

	protoTeams := make([]*Team, len(teams))
	for i, team := range teams {
		protoTeams[i] = convertTeamToProto(team)
	}

	return &GetAllTeamsResponse{
		Teams: protoTeams,
	}, nil
}

func (s *teamsServer) AddTeamMember(ctx context.Context, req *AddTeamMemberRequest) (*AddTeamMemberResponse, error) {
	if req.TeamId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "team ID cannot be empty")
	}
	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user ID cannot be empty")
	}
	if req.Role == Role_ROLE_UNKNOWN {
		return nil, status.Errorf(codes.InvalidArgument, "role cannot be UNKNOWN")
	}

	member := model.TeamMember{
		UserID: req.UserId,
		Role:   model.Role(req.Role),
	}

	if err := s.model.AddMember(req.TeamId, member); err != nil {
		if err.Error() == "team not found" {
			return nil, status.Errorf(codes.NotFound, "team not found")
		}
		if err.Error() == "user already a member of the team" {
			return nil, status.Errorf(codes.AlreadyExists, "user is already a member of the team")
		}
		return nil, status.Errorf(codes.Internal, "failed to add team member: %v", err)
	}

	return &AddTeamMemberResponse{}, nil
}

func (s *teamsServer) RemoveTeamMember(ctx context.Context, req *RemoveTeamMemberRequest) (*RemoveTeamMemberResponse, error) {
	if req.TeamId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "team ID cannot be empty")
	}
	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user ID cannot be empty")
	}

	if err := s.model.RemoveMember(req.TeamId, req.UserId); err != nil {
		if err.Error() == "team not found" {
			return nil, status.Errorf(codes.NotFound, "team not found")
		}
		if err.Error() == "user not found in team" {
			return nil, status.Errorf(codes.NotFound, "user is not a member of the team")
		}
		return nil, status.Errorf(codes.Internal, "failed to remove team member: %v", err)
	}

	return &RemoveTeamMemberResponse{}, nil
}

func (s *teamsServer) UpdateTeamMemberRole(ctx context.Context, req *UpdateTeamMemberRoleRequest) (*UpdateTeamMemberRoleResponse, error) {
	if req.TeamId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "team ID cannot be empty")
	}
	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user ID cannot be empty")
	}
	if req.Role == Role_ROLE_UNKNOWN {
		return nil, status.Errorf(codes.InvalidArgument, "role cannot be UNKNOWN")
	}

	if err := s.model.UpdateMemberRole(req.TeamId, req.UserId, model.Role(req.Role)); err != nil {
		if err.Error() == "team not found" {
			return nil, status.Errorf(codes.NotFound, "team not found")
		}
		if err.Error() == "user not found in team" {
			return nil, status.Errorf(codes.NotFound, "user is not a member of the team")
		}
		return nil, status.Errorf(codes.Internal, "failed to update team member role: %v", err)
	}

	return &UpdateTeamMemberRoleResponse{}, nil
}

func convertTeamToProto(team *model.Team) *Team {
	members := make([]*TeamMember, len(team.Members))
	for i, member := range team.Members {
		members[i] = &TeamMember{
			UserId: member.UserID,
			Role:   Role(member.Role),
		}
	}

	return &Team{
		Id:        team.ID,
		Name:      team.Name,
		Members:   members,
		CreatedAt: team.CreatedAt.Format(time.RFC3339),
	}
}
