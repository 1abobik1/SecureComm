package dto


// HandshakeFinalizeReq описывает запрос на завершение Handshake.
// swagger:model HandshakeFinalizeReq
// @description Клиент шлёт RSA-OAEP(encrypted payload), закодированный в Base64.
// @description Подробнее про поле encrypted... 
// @description Рандомные 32 байта - это сессионная строка, назовем её ks, которая лежит в payload
// @description payload - это сумма байтов (ks || nonce3 || nonce2)
// @description signature3 - это подписанный payload приватным ключем клиента
// @description В конце encrypted это зашифрованные байты (payload || signature3(в DER формате))
// @description encrypted - зашифрован RSA-OAEP публичным ключем сервера, отдается в формате Base64
type HandshakeFinalizeReq struct {
    // Base64(RSA-OAEP(encrypted payload || signature3(DER)))
    Encrypted string `json:"encrypted"`
    Signature3 string `json:"signature3"`
}

// HandshakeFinalizeResp описывает ответ на завершение Handshake.
// swagger:model HandshakeFinalizeResp
// @description Сервер возвращает подпись h4 = SHA256(Ks || nonce3 || nonce2), подписанную приватным ECDSA‑ключом сервера и закодированную в Base64.
type HandshakeFinalizeResp struct {
    // Base64(DER‑подпись ответа сервера)
    Signature4 string `json:"signature4"`
}
