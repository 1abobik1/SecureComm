package dto

type UserUsage struct {
	CurrentUsedGB  int    `json:"current_used_gb"`
	CurrentUsedMB  int    `json:"current_used_mb"`
	CurrentUsedKB  int    `json:"current_used_kb"`
	StorageLimitGB int    `json:"storage_limit_gb"`
	PlanName       string `json:"plan_name"`
}
