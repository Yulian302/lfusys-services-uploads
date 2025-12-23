package uploads

type UploadResponse struct {
	UploadId string `json:"upload_id" example:"abc123"`
	ChunkId  uint32 `json:"chunk_id" example:"1"`
	S3Key    string `json:"s3_key" example:"uploads/abc123/chunk_1"`
}
