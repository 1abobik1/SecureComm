package dto

// SignUpDTO для /user/signup
type SignUpDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Platform string `json:"platform"`
}
