package handshake_handler

import "context"

// интерфейс бизнес-логики handshake
type Service interface {
	Init(ctx context.Context, clientID string, clientRSA, clientECDSA []byte, nonce1 []byte, sig1 []byte) (serverRSA, serverECDSA, nonce2, signature2 []byte, er error)
	ComputeFingerprint(ctx context.Context, rsaPub, ecdsaPub []byte) string
	Finalize(ctx context.Context, clientID string, sig3, encrypted []byte) (signature4 []byte, er error)
	DecryptWithSession(ctx context.Context, clientID string, signature, blob []byte) ([]byte, error)
}

type HSHandler struct {
	svc Service
}

func NewHandler(svc Service) *HSHandler {
	return &HSHandler{svc: svc}
}
