package service

import (
	"context"
)

// DecryptWithSession расшифровывает пакет:
func (s *service) DecryptWithSession(ctx context.Context, clientID string, signature, blob []byte) ([]byte, error) {
	return s.parseSessionBlob(ctx, clientID, signature, blob)
}
