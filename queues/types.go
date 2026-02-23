package queues

type UploadCompleteMessage struct {
	UploadId string `json:"upload_id" binding:"required"`
	ChunkIdx uint32 `json:"chunk_idx" binding:"required"`
}
