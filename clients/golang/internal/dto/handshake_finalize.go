package dto

type FinalizeReq struct {
	Encrypted  string `json:"encrypted"`
	Signature3 string `json:"signature3"`
}

type FinalizeResp struct {
	Signature4 string `json:"signature4"`
}
