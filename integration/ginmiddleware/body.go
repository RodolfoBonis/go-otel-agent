package ginmiddleware

import (
	"bytes"

	"github.com/gin-gonic/gin"
)

// BodyLogWriter is a custom writer for capturing HTTP response body content.
type BodyLogWriter struct {
	gin.ResponseWriter
	Body *bytes.Buffer
}

// Write writes data to both the buffer and the underlying ResponseWriter.
func (w *BodyLogWriter) Write(b []byte) (int, error) {
	w.Body.Write(b)
	return w.ResponseWriter.Write(b)
}

// NewBodyLogWriter wraps a gin.ResponseWriter to capture the response body.
func NewBodyLogWriter(w gin.ResponseWriter) *BodyLogWriter {
	return &BodyLogWriter{
		ResponseWriter: w,
		Body:           &bytes.Buffer{},
	}
}
