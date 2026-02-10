package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	ratelimit "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterUserRoutes 注册用户相关路由（需要认证）
func RegisterUserRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	redisClient *redis.Client,
) {
	rateLimiter := ratelimit.NewRateLimiter(redisClient)
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	{
		// 用户接口
		user := authenticated.Group("/user")
		{
			user.GET("/profile", h.User.GetProfile)
			user.PUT("/password", h.User.ChangePassword)
			user.PUT("", h.User.UpdateProfile)
			user.GET("/invite-count", h.Agent.GetMyInviteCount)
			user.GET("/agent-contact", h.Agent.GetMyAgentContact)

			// 用户属性（联系方式等）
			attrs := user.Group("/attributes")
			{
				attrs.GET("/definitions", h.User.GetAttributeDefinitions)
				attrs.GET("", h.User.GetMyAttributes)
				attrs.PUT("", h.User.UpdateMyAttributes)
			}

			// TOTP 双因素认证
			totp := user.Group("/totp")
			{
				totp.GET("/status", h.Totp.GetStatus)
				totp.POST("/setup", h.Totp.InitiateSetup)
				totp.POST("/enable", h.Totp.Enable)
				totp.POST("/disable", h.Totp.Disable)
			}

			// 微信绑定/解绑
			wechat := user.Group("/wechat")
			{
				wechat.GET("/status", h.WeChatBinding.GetStatus)
				wechat.POST("/bind", h.WeChatBinding.InitiateBind)
				wechat.POST("/confirm", h.WeChatBinding.ConfirmBind)
				wechat.POST("/unbind", rateLimiter.LimitWithOptions("wechat-unbind", 5, time.Minute, ratelimit.RateLimitOptions{
					FailureMode: ratelimit.RateLimitFailClose,
				}), h.WeChatBinding.Unbind)
			}
		}

		// API Key管理
		keys := authenticated.Group("/keys")
		{
			keys.GET("", h.APIKey.List)
			keys.GET("/:id", h.APIKey.GetByID)
			keys.POST("", h.APIKey.Create)
			keys.PUT("/:id", h.APIKey.Update)
			keys.DELETE("/:id", h.APIKey.Delete)
		}

		// 用户可用分组（非管理员接口）
		groups := authenticated.Group("/groups")
		{
			groups.GET("/available", h.APIKey.GetAvailableGroups)
			groups.GET("/:id/models", h.APIKey.GetGroupAvailableModels)
			groups.GET("/rates", h.APIKey.GetUserGroupRates)
		}

		// 使用记录
		usage := authenticated.Group("/usage")
		{
			usage.GET("", h.Usage.List)
			usage.GET("/:id", h.Usage.GetByID)
			usage.GET("/stats", h.Usage.Stats)
			// User dashboard endpoints
			usage.GET("/dashboard/stats", h.Usage.DashboardStats)
			usage.GET("/dashboard/trend", h.Usage.DashboardTrend)
			usage.GET("/dashboard/models", h.Usage.DashboardModels)
			usage.POST("/dashboard/api-keys-usage", h.Usage.DashboardAPIKeysUsage)
		}

		// 公告（用户可见）
		announcements := authenticated.Group("/announcements")
		{
			announcements.GET("", h.Announcement.List)
			announcements.POST("/:id/read", h.Announcement.MarkRead)
		}

		// 卡密兑换
		redeem := authenticated.Group("/redeem")
		{
			redeem.POST("", h.Redeem.Redeem)
			redeem.GET("/history", h.Redeem.GetHistory)
		}

		// 用户订阅
		subscriptions := authenticated.Group("/subscriptions")
		{
			subscriptions.GET("", h.Subscription.List)
			subscriptions.GET("/active", h.Subscription.GetActive)
			subscriptions.GET("/progress", h.Subscription.GetProgress)
			subscriptions.GET("/summary", h.Subscription.GetSummary)
		}

		// 代理中心（用户端）
		agent := authenticated.Group("/agent")
		{
			agent.GET("/downline", h.Agent.GetMyDownline)
			agent.GET("/stats", h.Agent.GetMyInviteStats)
		}
	}
}
