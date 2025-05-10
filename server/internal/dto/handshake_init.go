package dto

type HandshakeReq struct {
	RSAPubClient   string `json:"rsa_pub_client"`
	ECDSAPubClient string `json:"ecdsa_pub_client"`
	Nonce1         string `json:"nonce1"`
	Signature1     string `json:"signature1"`
}

type HandshakeResp struct {
	ClientID       string `json:"client_id"`
	RSAPubServer   string `json:"rsa_pub_server"`
	ECDSAPubServer string `json:"ecdsa_pub_server"`
	Nonce2         string `json:"nonce2"`
	Signature2     string `json:"signature2"`
}
