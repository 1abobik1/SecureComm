package handshake_service

import (
	"crypto/hmac"
	"crypto/sha256"
	"math/big"
)

type der struct {
	R *big.Int
	S *big.Int
}

type FinalizePayload struct {
	Ks     []byte // 32 байта
	Nonce3 []byte // 8 байт
	Nonce2 []byte // 8 байт, от клиента
}

//Деривация K_enc и K_mac, поиск происходит по info параметру для K_enc (info=enc) для K_mac(info=mac)
func hkdfSha256(secret, info []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(info)
	return h.Sum(nil)[:32] // усечём до 32 байт
}
