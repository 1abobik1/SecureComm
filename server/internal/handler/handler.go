package handler

import "context"

// интерфейс бизнес-логики handshake
type Service interface {
	Init(ctx context.Context, clientID string, clientRSA, clientECDSA []byte, nonce1 []byte, sig1 []byte) (serverRSA, serverECDSA, nonce2, signature2 []byte, err error)
	ComputeFingerprint(ctx context.Context, rsaPub, ecdsaPub []byte) string
	Finalize(ctx context.Context, clientID string, encrypted []byte) (signature4 []byte, err error)
}

type handler struct {
	svc Service
}

func NewHandler(svc Service) *handler {
	return &handler{svc: svc}
}
