package dto

// @Description Bad request
type BadRequest struct {
	Error string `json:"error" example:"invalid request data"`
}