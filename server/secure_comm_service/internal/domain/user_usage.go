package domain

type UserUsage struct {
	CurrentUsed  int64  `db:"current_used" json:"current_used"`   // занято байт
	StorageLimit int64  `db:"storage_limit" json:"storage_limit"` // лимит байт
	PlanName     string `db:"plan_name"    json:"plan_name"`      // например "free" или "pro"
}
