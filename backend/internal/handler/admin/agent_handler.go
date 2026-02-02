package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AgentHandler handles admin agent management
type AgentHandler struct {
	agentService *service.AgentService
}

// NewAgentHandler creates a new admin agent handler
func NewAgentHandler(agentService *service.AgentService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
	}
}

// agentPaginationToResponse converts pagination.PaginationResult to response.PaginationResult
func agentPaginationToResponse(p *pagination.PaginationResult) *response.PaginationResult {
	if p == nil {
		return nil
	}
	return &response.PaginationResult{
		Total:    p.Total,
		Page:     p.Page,
		PageSize: p.PageSize,
		Pages:    p.Pages,
	}
}

// SetAgentRequest represents the request to set/unset agent status
type SetAgentRequest struct {
	IsAgent       bool   `json:"is_agent"`
	ParentAgentID *int64 `json:"parent_agent_id"`
}

// List handles listing all agents with pagination
// GET /api/v1/admin/agents
func (h *AgentHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)

	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 100 {
		search = search[:100]
	}

	agents, pagination, err := h.agentService.ListAgents(c.Request.Context(), page, pageSize, search)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	agentDTOs := make([]*dto.User, 0, len(agents))
	for i := range agents {
		agentDTOs = append(agentDTOs, dto.UserFromService(&agents[i]))
	}

	response.PaginatedWithResult(c, agentDTOs, agentPaginationToResponse(pagination))
}

// GetByID handles getting a single agent by ID
// GET /api/v1/admin/agents/:id
func (h *AgentHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid agent ID")
		return
	}

	agent, err := h.agentService.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.UserFromService(agent))
}

// SetAgentStatus handles setting/unsetting agent status for a user
// PATCH /api/v1/admin/agents/:id/status
func (h *AgentHandler) SetAgentStatus(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req SetAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	user, err := h.agentService.SetAgentStatus(c.Request.Context(), userID, req.IsAgent, req.ParentAgentID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.UserFromService(user))
}

// GetDownline handles getting downline users for an agent
// GET /api/v1/admin/agents/:id/downline
func (h *AgentHandler) GetDownline(c *gin.Context) {
	agentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid agent ID")
		return
	}

	page, pageSize := response.ParsePagination(c)

	users, pagination, err := h.agentService.GetDownlineUsers(c.Request.Context(), agentID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	userDTOs := make([]*dto.User, 0, len(users))
	for i := range users {
		userDTOs = append(userDTOs, dto.UserFromService(&users[i]))
	}

	response.PaginatedWithResult(c, userDTOs, agentPaginationToResponse(pagination))
}

// GetInviteStats handles getting invite statistics for an agent
// GET /api/v1/admin/agents/:id/invite-stats
func (h *AgentHandler) GetInviteStats(c *gin.Context) {
	agentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid agent ID")
		return
	}

	stats, err := h.agentService.GetInviteStats(c.Request.Context(), agentID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, stats)
}
