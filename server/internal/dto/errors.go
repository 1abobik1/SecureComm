package dto

// BadRequestErr описывает ответ с кодом 400.
// swagger:model BadRequestErr
type BadRequestErr struct {
	// example: invalid request data
	Error string `json:"error"`
}

// InternalServerErr описывает ответ с кодом 500.
// swagger:model InternalServerErr
type InternalServerErr struct {
	Error string `json:"error"`
}

// ConflictErr описывает ответ с кодом 409 (replay detected).
// swagger:model ConflictErr
type ConflictErr struct {
	// example: replay detected
	Error string `json:"error"`
}

// UnauthorizedErr описывает ответ с кодом 401.
// swagger:model UnauthorizedErr
type UnauthorizedErr struct {
	Error string `json:"error"`
}