package dto

// HandshakeInitReq описывает запрос на инициализацию Handshake.
// swagger:model HandshakeInitReq
// @description Клиент отправляет свои публичные ключи и nonce1, всё это подписано приватным ECDSA‑ключом.
// @description Все бинарные данные (ключи, подписи, nonce) закодированы в Base64 (DER для ключей и подписи).
// @description
// @description rsa_pub_client - Base64(DER‑закодированный RSA‑публичный ключ клиента)
// @description ecdsa_pub_client - Base64(DER‑закодированный ECDSA‑публичный ключ клиента)
// @description nonce1 - Base64(8‑байтовый случайный nonce)
// @description signature1 - Base64(DER‑закодированная подпись SHA256(clientRSA || clientECDSA || nonce1) приватным ECDSA‑ключом клиента)
type HandshakeInitReq struct {
	RSAPubClient   string `json:"rsa_pub_client"`
	ECDSAPubClient string `json:"ecdsa_pub_client"`
	Nonce1         string `json:"nonce1"`
	Signature1     string `json:"signature1"`
}

// HandshakeInitResp описывает ответ на инициализацию Handshake.
// swagger:model HandshakeInitResp
// @description Сервер отвечает своими публичными ключами и nonce2, всё это подписано приватным ECDSA‑ключом сервера.
// @description Все бинарные данные (ключи, подписи, nonce) закодированы в Base64 (DER для ключей и подписи).
// @description  
// @description client_id - SHA256‑хэш от (clientRSA‖clientECDSA), представлен в hex
// @description rsa_pub_server - Base64(DER‑закодированный RSA‑публичный ключ сервера)
// @description ecdsa_pub_server - Base64(DER‑закодированный ECDSA‑публичный ключ сервера)
// @description nonce2 - Base64(8‑байтовый случайный nonce)
// @description signature2 - Base64(DER‑подпись SHA256(rsaServer || ecdsaServer || nonce2 || nonce1 || clientID) приватным ECDSA‑ключом сервера)
type HandshakeInitResp struct {
	ClientID       string `json:"client_id"`
	RSAPubServer   string `json:"rsa_pub_server"`
	ECDSAPubServer string `json:"ecdsa_pub_server"`
	Nonce2         string `json:"nonce2"`
	Signature2     string `json:"signature2"`
}
