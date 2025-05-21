package models

type UserModel struct {
	ID          int
	Email       string
	Password    []byte
	IsActivated bool
}
