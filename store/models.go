package store

type UploadSession struct {
	TotalChunks    uint32 `dynamodbav:"total_chunks"`              // Number of 5MB chunks required
	UploadedChunks []int  `dynamodbav:"uploaded_chunks,omitempty"` // Bitmask of uploaded chunks (in bytes)
}
