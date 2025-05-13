package dto

type HandshakeFinalizeReq struct {
	Encrypted string `json:"encrypted"`
}

type HandshakeFinalizeResp struct {
	Signature4 string `json:"signature4"`
}
