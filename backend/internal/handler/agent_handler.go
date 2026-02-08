package handler

import (
	"log"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AgentHandler handles user-facing agent center endpoints
type AgentHandler struct {
	agentService *service.AgentService
	userService  *service.UserService
	attrService  *service.UserAttributeService
}

// NewAgentHandler creates a new user-facing agent handler
func NewAgentHandler(agentService *service.AgentService, userService *service.UserService, attrService *service.UserAttributeService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
		userService:  userService,
		attrService:  attrService,
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

// GetMyDownline handles getting downline users for the current agent
// GET /api/v1/agent/downline
func (h *AgentHandler) GetMyDownline(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	userID := subject.UserID

	// Check if user is an agent
	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !user.IsAgent {
		response.Forbidden(c, "You are not an agent")
		return
	}

	page, pageSize := response.ParsePagination(c)

	users, pag, err := h.agentService.GetDownlineUsers(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	userDTOs := make([]*dto.User, 0, len(users))
	for i := range users {
		userDTOs = append(userDTOs, dto.UserFromService(&users[i]))
	}

	response.PaginatedWithResult(c, userDTOs, agentPaginationToResponse(pag))
}

// GetMyInviteStats handles getting invite statistics for the current agent
// GET /api/v1/agent/stats
func (h *AgentHandler) GetMyInviteStats(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	userID := subject.UserID

	// Check if user is an agent
	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !user.IsAgent {
		response.Forbidden(c, "You are not an agent")
		return
	}

	stats, err := h.agentService.GetInviteStats(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, stats)
}

// GetMyInviteCount handles getting invite count for the current user (all users)
// GET /api/v1/user/invite-count
func (h *AgentHandler) GetMyInviteCount(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	userID := subject.UserID

	count, err := h.agentService.GetUserInviteCount(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, map[string]int{"invite_count": count})
}

// GetMyAgentContact handles getting the contact info of the user's assigned agent
// GET /api/v1/user/agent-contact
func (h *AgentHandler) GetMyAgentContact(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	userID := subject.UserID

	// Get current user to find their belong_agent_id
	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Determine which contact to show: agent or admin
	var contactUser *service.User
	var contactUserID int64
	var isAgent bool

	if user.IsAgent {
		// Agent views their own contact info
		contactUser = user
		contactUserID = user.ID
		isAgent = true
	} else if user.BelongAgentID != nil && *user.BelongAgentID != 0 {
		// User has an assigned agent; validate before using.
		agent, err := h.userService.GetByID(c.Request.Context(), *user.BelongAgentID)
		if err == nil && agent.IsActive() && agent.IsAgent {
			contactUser = agent
			contactUserID = agent.ID
			isAgent = true
		} else if err != nil {
			log.Printf("[AgentContact] failed to lookup agent %d for user %d, falling back to admin", *user.BelongAgentID, userID)
		}
		// Fall through to admin fallback if agent is invalid/inactive/revoked.
	}

	if contactUser == nil {
		admin, err := h.userService.GetFirstAdmin(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		contactUser = admin
		contactUserID = admin.ID
		isAgent = false
	}

	// Get contact user's attributes (contact info like WeChat, QQ, etc.)
	attrValues, err := h.attrService.GetUserAttributes(c.Request.Context(), contactUserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Get attribute definitions to include names
	attrDefs, err := h.attrService.ListDefinitions(c.Request.Context(), true)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Build attribute map with definition info
	defMap := make(map[int64]service.UserAttributeDefinition)
	for _, def := range attrDefs {
		defMap[def.ID] = def
	}

	// Build attributes response with name and value
	attributes := make([]map[string]interface{}, 0)
	for _, val := range attrValues {
		if def, ok := defMap[val.AttributeID]; ok && val.Value != "" {
			attributes = append(attributes, map[string]interface{}{
				"key":   def.Key,
				"name":  def.Name,
				"type":  string(def.Type),
				"value": val.Value,
			})
		}
	}

	// Return contact info with attributes
	response.Success(c, map[string]interface{}{
		"has_agent": isAgent,
		"agent": map[string]interface{}{
			"email":      contactUser.Email,
			"username":   contactUser.Username,
			"attributes": attributes,
		},
	})
}
