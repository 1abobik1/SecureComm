package external_api

import (
	"net/http"
)

type WEBClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewWEBClient(baseURL string, httpClient *http.Client) *WEBClient {
	return &WEBClient{baseURL: baseURL, httpClient: httpClient}
}

type TGClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewTGClient(baseURL string, httpClient *http.Client) *TGClient {
	return &TGClient{baseURL: baseURL, httpClient: httpClient}
}
