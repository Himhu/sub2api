package dto

// SystemSettings represents the admin settings API response payload.
type SystemSettings struct {
	RegistrationEnabled         bool `json:"registration_enabled"`
	InviteRegistrationEnabled   bool `json:"invite_registration_enabled"`
	TotpEnabled                 bool `json:"totp_enabled"`                   // TOTP 双因素认证
	TotpEncryptionKeyConfigured bool `json:"totp_encryption_key_configured"` // TOTP 加密密钥是否已配置

	TurnstileEnabled             bool   `json:"turnstile_enabled"`
	TurnstileSiteKey             string `json:"turnstile_site_key"`
	TurnstileSecretKeyConfigured bool   `json:"turnstile_secret_key_configured"`

	LinuxDoConnectEnabled                bool   `json:"linuxdo_connect_enabled"`
	LinuxDoConnectClientID               string `json:"linuxdo_connect_client_id"`
	LinuxDoConnectClientSecretConfigured bool   `json:"linuxdo_connect_client_secret_configured"`
	LinuxDoConnectRedirectURL            string `json:"linuxdo_connect_redirect_url"`

	SiteName                    string `json:"site_name"`
	SiteLogo                    string `json:"site_logo"`
	SiteSubtitle                string `json:"site_subtitle"`
	APIBaseURL                  string `json:"api_base_url"`
	DocURL                      string `json:"doc_url"`
	HomeContent                 string `json:"home_content"`
	HideCcsImportButton         bool   `json:"hide_ccs_import_button"`
	PurchaseSubscriptionEnabled bool   `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL     string `json:"purchase_subscription_url"`

	DefaultConcurrency int     `json:"default_concurrency"`
	DefaultBalance     float64 `json:"default_balance"`
	InviterBonus       float64 `json:"inviter_bonus"`  // 邀请人奖励余额
	InviteeBonus       float64 `json:"invitee_bonus"`  // 被邀请人奖励余额

	// Model fallback configuration
	EnableModelFallback      bool   `json:"enable_model_fallback"`
	FallbackModelAnthropic   string `json:"fallback_model_anthropic"`
	FallbackModelOpenAI      string `json:"fallback_model_openai"`
	FallbackModelGemini      string `json:"fallback_model_gemini"`
	FallbackModelAntigravity string `json:"fallback_model_antigravity"`

	// Identity patch configuration (Claude -> Gemini)
	EnableIdentityPatch bool   `json:"enable_identity_patch"`
	IdentityPatchPrompt string `json:"identity_patch_prompt"`

	// Ops monitoring (vNext)
	OpsMonitoringEnabled         bool   `json:"ops_monitoring_enabled"`
	OpsRealtimeMonitoringEnabled bool   `json:"ops_realtime_monitoring_enabled"`
	OpsQueryModeDefault          string `json:"ops_query_mode_default"`
	OpsMetricsIntervalSeconds    int    `json:"ops_metrics_interval_seconds"`

	// WeChat Service Account
	WeChatEnabled             bool   `json:"wechat_enabled"`
	WeChatAppID               string `json:"wechat_app_id"`
	WeChatAppSecretConfigured bool   `json:"wechat_app_secret_configured"`
	WeChatTokenConfigured     bool   `json:"wechat_token_configured"`
	WeChatAccountName         string `json:"wechat_account_name"`
}

type PublicSettings struct {
	RegistrationEnabled       bool `json:"registration_enabled"`
	InviteRegistrationEnabled bool `json:"invite_registration_enabled"`
	TotpEnabled                 bool   `json:"totp_enabled"` // TOTP 双因素认证
	TurnstileEnabled            bool   `json:"turnstile_enabled"`
	TurnstileSiteKey            string `json:"turnstile_site_key"`
	SiteName                    string `json:"site_name"`
	SiteLogo                    string `json:"site_logo"`
	SiteSubtitle                string `json:"site_subtitle"`
	APIBaseURL                  string `json:"api_base_url"`
	DocURL                      string `json:"doc_url"`
	HomeContent                 string `json:"home_content"`
	HideCcsImportButton         bool   `json:"hide_ccs_import_button"`
	PurchaseSubscriptionEnabled bool   `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL     string `json:"purchase_subscription_url"`
	LinuxDoOAuthEnabled         bool   `json:"linuxdo_oauth_enabled"`
	Version                     string `json:"version"`
	// Invite bonus settings
	InviterBonus float64 `json:"inviter_bonus"` // 邀请人奖励余额
	InviteeBonus float64 `json:"invitee_bonus"` // 被邀请人奖励余额

	WeChatEnabled     bool   `json:"wechat_enabled"`
	WeChatAccountName string `json:"wechat_account_name"`
}

// StreamTimeoutSettings 流超时处理配置 DTO
type StreamTimeoutSettings struct {
	Enabled                bool   `json:"enabled"`
	Action                 string `json:"action"`
	TempUnschedMinutes     int    `json:"temp_unsched_minutes"`
	ThresholdCount         int    `json:"threshold_count"`
	ThresholdWindowMinutes int    `json:"threshold_window_minutes"`
}
