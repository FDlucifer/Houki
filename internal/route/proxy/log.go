package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wuhan005/Houki/internal/sse"
)

const pingInterval = time.Second * 30

func LogHandler(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	f, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	_, _ = io.WriteString(c.Writer, ": ping\n\n")
	f.Flush()

	ctx, cancel := context.WithCancel(c)
	defer cancel()
	events, errC := sse.GetStream().Tail(ctx)
	_, _ = io.WriteString(c.Writer, "events: stream opened\n\n")
	f.Flush()

L:
	for {
		select {
		case <-ctx.Done():
			_, _ = io.WriteString(c.Writer, "events: stream cancelled\n\n")
			f.Flush()
			break L
		case <-errC:
			_, _ = io.WriteString(c.Writer, "events: stream error\n\n")
			f.Flush()
			break L
		case <-time.After(time.Hour):
			_, _ = io.WriteString(c.Writer, "events: stream timeout\n\n")
			f.Flush()
			break L
		case <-time.After(pingInterval):
			_, _ = io.WriteString(c.Writer, ": ping\n\n")
			f.Flush()
		case event := <-events:
			_, _ = io.WriteString(c.Writer, "data: ")
			evt, err := json.Marshal(event)
			if err != nil {
				continue
			}
			_, _ = c.Writer.Write(evt)
			_, _ = io.WriteString(c.Writer, "\n\n")
			f.Flush()
		}
	}

	_, _ = io.WriteString(c.Writer, "event: error\ndata: eof\n\n")
	f.Flush()
	_, _ = io.WriteString(c.Writer, "events: stream closed")
	f.Flush()
}
