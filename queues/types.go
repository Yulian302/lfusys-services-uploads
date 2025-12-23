package queues

type UploadCompleteMessage struct {
	UploadId string `json:"upload_id" binding:"required"`
}
