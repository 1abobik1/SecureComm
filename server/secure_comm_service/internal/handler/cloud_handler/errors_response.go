package cloud_handler

// ErrorResponse описывает единый формат ответа с ошибкой.
// swagger:model ErrorResponse
type ErrorResponse struct {
    // Код HTTP-статуса
    // example: 400
    Status int `json:"status"`
    // Краткое описание ошибки
    // example: "Invalid request"
    Error string `json:"error"`
    // Дополнительные детали (при наличии)
    // example: "field 'file' is required"
    Details interface{} `json:"details,omitempty"`
}