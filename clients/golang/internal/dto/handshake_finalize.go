package dto

type FinalizeReq struct {
	Encrypted string `json:"encrypted"`
}

type FinalizeResp struct {
	Signature4 string `json:"signature4"`
}