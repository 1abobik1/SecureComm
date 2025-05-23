package domain

import "time"

type FileContent struct {
	Name      string
	Format    string
	CreatedAt time.Time
	Data      []byte
}
