package dto

// SessionMessageReq — зашифрованное сообщение с metadata.
// swagger:model SessionMessageReq
type SessionMessageReq struct {
	// Base64(IV || ciphertext || tag)
	EncryptedMessage string `json:"encrypted_message"`
	ClientSignature  string `json:"client_signature"`
}

// SessionMessageResp — расшифрованный ответ.
// swagger:model SessionMessageResp
type SessionMessageResp struct {
	Plaintext string `json:"plaintext"`
}