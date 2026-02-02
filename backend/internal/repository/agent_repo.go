package repository

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type agentRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

// NewAgentRepository creates a new agent repository
func NewAgentRepository(client *dbent.Client, sqlDB *sql.DB) service.AgentRepository {
	return &agentRepository{client: client, sql: sqlDB}
}

// ListAgents returns a paginated list of agents (users with is_agent=true)
func (r *agentRepository) ListAgents(ctx context.Context, params pagination.PaginationParams, search string) ([]service.User, *pagination.PaginationResult, error) {
	q := r.client.User.Query().Where(dbuser.IsAgentEQ(true))

	if search != "" {
		q = q.Where(
			dbuser.Or(
				dbuser.EmailContainsFold(search),
				dbuser.UsernameContainsFold(search),
			),
		)
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	users, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(dbuser.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	outUsers := make([]service.User, 0, len(users))
	for i := range users {
		u := userEntityToService(users[i])
		outUsers = append(outUsers, *u)
	}

	return outUsers, paginationResultFromTotal(int64(total), params), nil
}

// GetAgentByID returns an agent by ID
func (r *agentRepository) GetAgentByID(ctx context.Context, id int64) (*service.User, error) {
	m, err := r.client.User.Query().Where(dbuser.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	return userEntityToService(m), nil
}

// SetAgentStatus sets or unsets agent status for a user
func (r *agentRepository) SetAgentStatus(ctx context.Context, userID int64, isAgent bool, parentAgentID *int64, inviteCode *string) (*service.User, error) {
	// 使用事务确保数据一致性
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	// 确保事务在出错时回滚
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	txClient := tx.Client()
	update := txClient.User.UpdateOneID(userID).SetIsAgent(isAgent)

	if isAgent {
		if parentAgentID != nil {
			update = update.SetParentAgentID(*parentAgentID)
		}
		if inviteCode != nil {
			update = update.SetInviteCode(*inviteCode)
		}
	} else {
		// When revoking agent status, first get the agent's parent to reassign downline
		agent, err := txClient.User.Get(ctx, userID)
		if err != nil {
			return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
		}

		// 1. Update all users who belong to this agent (BelongAgentID)
		// Reassign them to the agent's parent (or null if no parent)
		downlineUpdate := txClient.User.Update().Where(dbuser.BelongAgentIDEQ(userID))
		if agent.ParentAgentID != nil {
			downlineUpdate = downlineUpdate.SetBelongAgentID(*agent.ParentAgentID)
		} else {
			downlineUpdate = downlineUpdate.ClearBelongAgentID()
		}
		if _, err := downlineUpdate.Save(ctx); err != nil {
			return nil, err
		}

		// 2. Update all child agents (ParentAgentID)
		// Reassign their parent to the revoked agent's parent
		childAgentUpdate := txClient.User.Update().Where(dbuser.ParentAgentIDEQ(userID))
		if agent.ParentAgentID != nil {
			childAgentUpdate = childAgentUpdate.SetParentAgentID(*agent.ParentAgentID)
		} else {
			childAgentUpdate = childAgentUpdate.ClearParentAgentID()
		}
		if _, err := childAgentUpdate.Save(ctx); err != nil {
			return nil, err
		}

		// Clear ParentAgentID but preserve InviteCode (avoid breaking published invite links)
		update = update.ClearParentAgentID()
	}

	m, err := update.Save(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return userEntityToService(m), nil
}

// GetDownlineUsers returns users invited by an agent
func (r *agentRepository) GetDownlineUsers(ctx context.Context, agentID int64, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	q := r.client.User.Query().Where(
		dbuser.Or(
			dbuser.InvitedByUserIDEQ(agentID),
			dbuser.BelongAgentIDEQ(agentID),
		),
	)

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	users, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(dbuser.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	outUsers := make([]service.User, 0, len(users))
	for i := range users {
		u := userEntityToService(users[i])
		outUsers = append(outUsers, *u)
	}

	return outUsers, paginationResultFromTotal(int64(total), params), nil
}

// GetInviteStats returns invite statistics for an agent
func (r *agentRepository) GetInviteStats(ctx context.Context, agentID int64) (*service.InviteStats, error) {
	// Count total invited users
	totalInvited, err := r.client.User.Query().
		Where(dbuser.InvitedByUserIDEQ(agentID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// Count invited agents
	invitedAgents, err := r.client.User.Query().
		Where(
			dbuser.InvitedByUserIDEQ(agentID),
			dbuser.IsAgentEQ(true),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// Count direct downline (belong_agent_id = agentID)
	directInvited, err := r.client.User.Query().
		Where(dbuser.BelongAgentIDEQ(agentID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return &service.InviteStats{
		AgentID:       agentID,
		TotalInvited:  totalInvited,
		InvitedAgents: invitedAgents,
		InvitedUsers:  totalInvited - invitedAgents,
		DirectInvited: directInvited,
	}, nil
}

// GetUserInviteCount returns the number of users invited by a user
func (r *agentRepository) GetUserInviteCount(ctx context.Context, userID int64) (int, error) {
	return r.client.User.Query().
		Where(dbuser.InvitedByUserIDEQ(userID)).
		Count(ctx)
}
