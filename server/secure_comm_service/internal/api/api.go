package api

import "github.com/1abobik1/SecureComm/internal/service/handshake_service"

type TGClientKeysAPI struct {
	sessionI handshake_service.SessionStore
}

func NewTGClientKeysAPI(sessionI handshake_service.SessionStore) *TGClientKeysAPI {
	return &TGClientKeysAPI{sessionI: sessionI}
}

type WEBClientKeysAPI struct {
	sessionI handshake_service.SessionStore
}

func NewWEBClientKeysAPI(sessionI handshake_service.SessionStore) *WEBClientKeysAPI {
	return &WEBClientKeysAPI{sessionI: sessionI}
}
