package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// AgentRepository defines the interface for agent-related database operations
type AgentRepository interface {
	ListAgents(ctx context.Context, params pagination.PaginationParams, search string) ([]User, *pagination.PaginationResult, error)
	GetAgentByID(ctx context.Context, id int64) (*User, error)
	SetAgentStatus(ctx context.Context, userID int64, isAgent bool, parentAgentID *int64, inviteCode *string) (*User, error)
	GetDownlineUsers(ctx context.Context, agentID int64, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error)
	GetInviteStats(ctx context.Context, agentID int64) (*InviteStats, error)
	GetUserInviteCount(ctx context.Context, userID int64) (int, error)
}

// InviteStats represents invite statistics for an agent
type InviteStats struct {
	AgentID        int64 `json:"agent_id"`
	TotalInvited   int   `json:"total_invited"`
	InvitedAgents  int   `json:"invited_agents"`
	InvitedUsers   int   `json:"invited_users"`
	DirectInvited  int   `json:"direct_invited"`
}

// AgentService handles agent-related business logic
type AgentService struct {
	agentRepo AgentRepository
	userRepo  UserRepository
}

// NewAgentService creates a new AgentService instance
func NewAgentService(agentRepo AgentRepository, userRepo UserRepository) *AgentService {
	return &AgentService{
		agentRepo: agentRepo,
		userRepo:  userRepo,
	}
}

// ListAgents returns a paginated list of agents
func (s *AgentService) ListAgents(ctx context.Context, page, pageSize int, search string) ([]User, *pagination.PaginationResult, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	return s.agentRepo.ListAgents(ctx, params, search)
}

// GetAgentByID returns an agent by ID
func (s *AgentService) GetAgentByID(ctx context.Context, id int64) (*User, error) {
	agent, err := s.agentRepo.GetAgentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !agent.IsAgent {
		return nil, ErrUserNotFound
	}
	return agent, nil
}

// SetAgentStatus sets or unsets agent status for a user
func (s *AgentService) SetAgentStatus(ctx context.Context, userID int64, isAgent bool, parentAgentID *int64) (*User, error) {
	var inviteCode *string

	// Validate parent agent if setting as agent with a parent
	if isAgent && parentAgentID != nil {
		// Check for self-reference
		if *parentAgentID == userID {
			return nil, ErrParentAgentSelf
		}
		// Verify parent is actually an agent
		parent, err := s.userRepo.GetByID(ctx, *parentAgentID)
		if err != nil {
			return nil, err
		}
		if !parent.IsAgent {
			return nil, ErrParentAgentInvalid
		}
		// Check for cycle in parent chain
		if err := s.detectParentCycle(ctx, userID, *parentAgentID); err != nil {
			return nil, err
		}
	}

	// Generate invite code if setting as agent (preserve existing code if user already has one)
	if isAgent {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		// Preserve existing invite code if user already has one (avoid breaking existing invite links)
		if user.InviteCode != nil && *user.InviteCode != "" {
			inviteCode = user.InviteCode
		} else {
			code, err := s.generateUniqueInviteCode(ctx)
			if err != nil {
				return nil, fmt.Errorf("generate invite code: %w", err)
			}
			inviteCode = &code
		}
	}

	return s.agentRepo.SetAgentStatus(ctx, userID, isAgent, parentAgentID, inviteCode)
}

// GetDownlineUsers returns users invited by an agent
func (s *AgentService) GetDownlineUsers(ctx context.Context, agentID int64, page, pageSize int) ([]User, *pagination.PaginationResult, error) {
	// Verify the user is actually an agent before fetching downline data
	if _, err := s.GetAgentByID(ctx, agentID); err != nil {
		return nil, nil, err
	}
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	return s.agentRepo.GetDownlineUsers(ctx, agentID, params)
}

// GetInviteStats returns invite statistics for an agent
func (s *AgentService) GetInviteStats(ctx context.Context, agentID int64) (*InviteStats, error) {
	// Verify the user is actually an agent before fetching stats
	if _, err := s.GetAgentByID(ctx, agentID); err != nil {
		return nil, err
	}
	return s.agentRepo.GetInviteStats(ctx, agentID)
}

// GetUserInviteCount returns the number of users invited by a user (for all users, not just agents)
func (s *AgentService) GetUserInviteCount(ctx context.Context, userID int64) (int, error) {
	return s.agentRepo.GetUserInviteCount(ctx, userID)
}

// generateInviteCode generates a unique invite code (uppercase)
func generateInviteCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// 转换为大写，与验证时的 ToUpper 保持一致
	return strings.ToUpper(hex.EncodeToString(bytes)), nil
}

// generateUniqueInviteCode generates a unique invite code with collision retry
func (s *AgentService) generateUniqueInviteCode(ctx context.Context) (string, error) {
	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		code, err := generateInviteCode()
		if err != nil {
			return "", err
		}
		// Check if code already exists
		_, err = s.userRepo.GetByInviteCode(ctx, code)
		if errors.Is(err, ErrUserNotFound) {
			// Code doesn't exist, we can use it
			return code, nil
		}
		if err != nil {
			// Database error, return it instead of masking
			return "", fmt.Errorf("check invite code existence: %w", err)
		}
		// Code exists, retry with a new one
	}
	return "", fmt.Errorf("failed to generate unique invite code after %d attempts", maxRetries)
}

// detectParentCycle checks if setting parentAgentID as parent of userID would create a cycle
// It traverses up the parent chain from parentAgentID and checks if userID is encountered
func (s *AgentService) detectParentCycle(ctx context.Context, userID, parentAgentID int64) error {
	visited := make(map[int64]bool)
	currentID := parentAgentID

	// Limit iterations to prevent infinite loops in case of existing bad data
	const maxDepth = 100
	for i := 0; i < maxDepth; i++ {
		if currentID == userID {
			return ErrParentAgentCycle
		}
		if visited[currentID] {
			// Already visited this node, there's an existing cycle in the data
			return ErrParentAgentCycle
		}
		visited[currentID] = true

		user, err := s.userRepo.GetByID(ctx, currentID)
		if errors.Is(err, ErrUserNotFound) {
			// User not found, reached the end of the chain, no cycle
			return nil
		}
		if err != nil {
			// Database error, return it instead of masking
			return fmt.Errorf("check parent cycle: %w", err)
		}
		if user.ParentAgentID == nil {
			// Reached the top of the chain, no cycle
			return nil
		}
		currentID = *user.ParentAgentID
	}
	// Exceeded max depth, likely a cycle or very deep hierarchy
	return ErrParentAgentCycle
}
