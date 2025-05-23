package dto

type ObjectIDs struct {
	ObjectIDs []ObjectID `json:"object_ids"`
}

type ObjectID struct {
	ObjID        string `json:"obj_id"`
	FileCategory string `json:"file_category"`
}

type FileResponse struct {
	Name       string `json:"name"`
	Created_At string `json:"created_at"`
	ObjID      string `json:"obj_id"`
	Url        string `json:"url"`
	MimeType   string `json:"mime_type"`
}
