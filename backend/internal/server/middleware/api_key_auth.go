package middleware

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// NewAPIKeyAuthMiddleware 创建 API Key 认证中间件
func NewAPIKeyAuthMiddleware(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) APIKeyAuthMiddleware {
	return APIKeyAuthMiddleware(apiKeyAuthWithSubscription(apiKeyService, subscriptionService, cfg))
}

// apiKeyAuthWithSubscription API Key认证中间件（支持订阅验证）
func apiKeyAuthWithSubscription(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		queryKey := strings.TrimSpace(c.Query("key"))
		queryApiKey := strings.TrimSpace(c.Query("api_key"))
		if queryKey != "" || queryApiKey != "" {
			AbortWithError(c, 400, "api_key_in_query_deprecated", "API key in query parameter is deprecated. Please use Authorization header instead.")
			return
		}

		// 尝试从Authorization header中提取API key (Bearer scheme)
		authHeader := c.GetHeader("Authorization")
		var apiKeyString string

		if authHeader != "" {
			// 验证Bearer scheme
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				apiKeyString = parts[1]
			}
		}

		// 如果Authorization header中没有，尝试从x-api-key header中提取
		if apiKeyString == "" {
			apiKeyString = c.GetHeader("x-api-key")
		}

		// 如果x-api-key header中没有，尝试从x-goog-api-key header中提取（Gemini CLI兼容）
		if apiKeyString == "" {
			apiKeyString = c.GetHeader("x-goog-api-key")
		}

		// 如果所有header都没有API key
		if apiKeyString == "" {
			AbortWithError(c, 401, "API_KEY_REQUIRED", "API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header")
			return
		}

		// 从数据库验证API key
		apiKey, err := apiKeyService.GetByKey(c.Request.Context(), apiKeyString)
		if err != nil {
			if errors.Is(err, service.ErrAPIKeyNotFound) {
				AbortWithError(c, 401, "INVALID_API_KEY", "Invalid API key")
				return
			}
			AbortWithError(c, 500, "INTERNAL_ERROR", "Failed to validate API key")
			return
		}

		// 检查API key是否激活
		if !apiKey.IsActive() {
			// Provide more specific error message based on status
			switch apiKey.Status {
			case service.StatusAPIKeyQuotaExhausted:
				AbortWithError(c, 429, "API_KEY_QUOTA_EXHAUSTED", "API key 额度已用完")
			case service.StatusAPIKeyExpired:
				AbortWithError(c, 403, "API_KEY_EXPIRED", "API key 已过期")
			default:
				AbortWithError(c, 401, "API_KEY_DISABLED", "API key is disabled")
			}
			return
		}

		// 检查API Key是否过期（即使状态是active，也要检查时间）
		if apiKey.IsExpired() {
			AbortWithError(c, 403, "API_KEY_EXPIRED", "API key 已过期")
			return
		}

		// 检查API Key配额是否耗尽
		if apiKey.IsQuotaExhausted() {
			AbortWithError(c, 429, "API_KEY_QUOTA_EXHAUSTED", "API key 额度已用完")
			return
		}

		// 检查 IP 限制（白名单/黑名单）
		// 注意：错误信息故意模糊，避免暴露具体的 IP 限制机制
		if len(apiKey.IPWhitelist) > 0 || len(apiKey.IPBlacklist) > 0 {
			clientIP := ip.GetClientIP(c)
			allowed, _ := ip.CheckIPRestriction(clientIP, apiKey.IPWhitelist, apiKey.IPBlacklist)
			if !allowed {
				AbortWithError(c, 403, "ACCESS_DENIED", "Access denied")
				return
			}
		}

		// 检查关联的用户
		if apiKey.User == nil {
			AbortWithError(c, 401, "USER_NOT_FOUND", "User associated with API key not found")
			return
		}

		// 检查用户状态
		if !apiKey.User.IsActive() {
			AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
			return
		}

		if cfg.RunMode == config.RunModeSimple {
			// 简易模式：跳过余额和订阅检查，但仍需设置必要的上下文
			c.Set(string(ContextKeyAPIKey), apiKey)
			c.Set(string(ContextKeyUser), AuthSubject{
				UserID:      apiKey.User.ID,
				Concurrency: apiKey.User.Concurrency,
			})
			c.Set(string(ContextKeyUserRole), apiKey.User.Role)
			setGroupContext(c, apiKey.Group)
			c.Next()
			return
		}

		subscriptionValidated := false

		if apiKey.SubscriptionID != nil && subscriptionService != nil {
			// 跨平台订阅模式：按 subscription_id 直接加载订阅，共享额度池
			subscription, err := subscriptionService.GetByID(c.Request.Context(), *apiKey.SubscriptionID)
			if err != nil {
				AbortWithError(c, 403, "SUBSCRIPTION_NOT_FOUND", "Bound subscription not found")
				return
			}
			if subscription.UserID != apiKey.User.ID {
				AbortWithError(c, 403, "SUBSCRIPTION_INVALID", "Subscription does not belong to this user")
				return
			}
			if err := subscriptionService.ValidateSubscription(c.Request.Context(), subscription); err != nil {
				AbortWithError(c, 403, "SUBSCRIPTION_INVALID", err.Error())
				return
			}
			subscriptionValidated = true

			if err := subscriptionService.CheckAndActivateWindow(c.Request.Context(), subscription); err != nil {
				log.Printf("Failed to activate subscription windows: %v", err)
			}
			if err := subscriptionService.CheckAndResetWindows(c.Request.Context(), subscription); err != nil {
				log.Printf("Failed to reset subscription windows: %v", err)
			}

			// 限额检查使用订阅所属分组的限额配置（非 API Key 的分组）
			limitGroup := subscription.Group
			if limitGroup == nil {
				AbortWithError(c, 500, "INTERNAL_ERROR", "Subscription group not loaded")
				return
			}
			if err := subscriptionService.CheckUsageLimits(c.Request.Context(), subscription, limitGroup, 0); err != nil {
				if apiKey.User.Points <= 0 {
					AbortWithError(c, 429, "USAGE_LIMIT_EXCEEDED", err.Error())
					return
				}
			}

			c.Set(string(ContextKeySubscription), subscription)
		} else if apiKey.Group != nil && apiKey.Group.IsSubscriptionType() && subscriptionService != nil {
			// 向后兼容：按 (user_id, group_id) 查找订阅
			subscription, err := subscriptionService.GetActiveSubscription(
				c.Request.Context(),
				apiKey.User.ID,
				apiKey.Group.ID,
			)
			if err != nil {
				AbortWithError(c, 403, "SUBSCRIPTION_NOT_FOUND", "No active subscription found for this group")
				return
			}
			if err := subscriptionService.ValidateSubscription(c.Request.Context(), subscription); err != nil {
				AbortWithError(c, 403, "SUBSCRIPTION_INVALID", err.Error())
				return
			}
			subscriptionValidated = true

			if err := subscriptionService.CheckAndActivateWindow(c.Request.Context(), subscription); err != nil {
				log.Printf("Failed to activate subscription windows: %v", err)
			}
			if err := subscriptionService.CheckAndResetWindows(c.Request.Context(), subscription); err != nil {
				log.Printf("Failed to reset subscription windows: %v", err)
			}
			if err := subscriptionService.CheckUsageLimits(c.Request.Context(), subscription, apiKey.Group, 0); err != nil {
				if apiKey.User.Points <= 0 {
					AbortWithError(c, 429, "USAGE_LIMIT_EXCEEDED", err.Error())
					return
				}
			}

			c.Set(string(ContextKeySubscription), subscription)
		} else {
			// 余额模式：检查用户余额和积分
			if apiKey.User.Balance <= 0 && apiKey.User.Points <= 0 {
				AbortWithError(c, 403, "INSUFFICIENT_BALANCE", "Insufficient account balance")
				return
			}
		}

		// 分组类型运行时检查（能力模型）
		if apiKey.Group != nil {
			canUsePointsOnly := apiKey.User.Points > 0
			canUseNormal := apiKey.User.Balance > 0 || subscriptionValidated
			if apiKey.Group.IsPointsOnly {
				if !canUsePointsOnly {
					AbortWithError(c, 403, "GROUP_TYPE_MISMATCH", "Points-only group requires points balance")
					return
				}
			} else {
				if !canUseNormal {
					AbortWithError(c, 403, "GROUP_TYPE_MISMATCH", "Normal group requires balance or valid subscription")
					return
				}
			}
		}

		// 将API key和用户信息存入上下文
		c.Set(string(ContextKeyAPIKey), apiKey)
		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: apiKey.User.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), apiKey.User.Role)
		setGroupContext(c, apiKey.Group)

		c.Next()
	}
}

// GetAPIKeyFromContext 从上下文中获取API key
func GetAPIKeyFromContext(c *gin.Context) (*service.APIKey, bool) {
	value, exists := c.Get(string(ContextKeyAPIKey))
	if !exists {
		return nil, false
	}
	apiKey, ok := value.(*service.APIKey)
	return apiKey, ok
}

// GetSubscriptionFromContext 从上下文中获取订阅信息
func GetSubscriptionFromContext(c *gin.Context) (*service.UserSubscription, bool) {
	value, exists := c.Get(string(ContextKeySubscription))
	if !exists {
		return nil, false
	}
	subscription, ok := value.(*service.UserSubscription)
	return subscription, ok
}

func setGroupContext(c *gin.Context, group *service.Group) {
	if !service.IsGroupContextValid(group) {
		return
	}
	if existing, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group); ok && existing != nil && existing.ID == group.ID && service.IsGroupContextValid(existing) {
		return
	}
	ctx := context.WithValue(c.Request.Context(), ctxkey.Group, group)
	c.Request = c.Request.WithContext(ctx)
}
