package model

type AuthAttempt struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	IP       string `json:"ip"`
}

type AuthResult struct {
	OK bool `json:"ok"`
}

type ResetRequest struct {
	Login string `json:"login"`
	IP    string `json:"ip"`
}

type NetworkListType string

const (
	ListTypeWhitelist NetworkListType = "whitelist"
	ListTypeBlacklist NetworkListType = "blacklist"
)
