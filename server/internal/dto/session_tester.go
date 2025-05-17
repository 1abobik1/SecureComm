package dto

// SessionTestReq — запрос на тестовое расшифрование сессионного сообщения.
// swagger:model SessionTestReq
// @description Клиент шлёт зашифрованное сессионным ключом сообщение в Base64.
type SessionTestReq struct {
    // Base64(ciphertext), полученный при шифровании сессии
    EncryptedMessage string `json:"encrypted_message"`
}

// SessionTestResp — ответ с расшифрованным текстом.
// swagger:model SessionTestResp
// @description Сервер вернул plaintext, расшифрованный текущим сессионным ключом.
type SessionTestResp struct {
    // Расшифрованный текст (UTF-8)
    Plaintext string `json:"plaintext"`
}