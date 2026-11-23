package middleware

import (
	"bufio"
	"bytes"
	"mime"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/clienterror"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// defaultClientErrorBodyLimit 是错误响应的最大缓冲字节数。
// 超过上限说明这个「错误体」不正常（上游可能塞了大段内容），直接丢弃原始内容、
// 只发网关文案——绝不为了省内存把原文漏给客户端。
const defaultClientErrorBodyLimit = 64 * 1024

// SanitizeUpstreamErrorBody 把「由上游错误导致的客户可见响应」统一重建：
// 丢弃上游自由文本（部分上游会在错误体里写加群/导流文案撬客户），
// 只保留白名单内的结构化标识（type / code / Google status）。
//
// 触发条件（三者同时满足才改写）：
//  1. HTTP 状态码 >= 400
//  2. 本次请求确实记录过上游错误（ops 上下文里有上游状态/消息/事件）；
//     这样网关自己的错误文案（余额不足、模型不在白名单等）保持原样。
//  3. Content-Type 是 JSON，且不是 text/event-stream
//
// 其余情况一律原样直通，正常流量、SSE 流式与 WebSocket 完全不受影响。
func SanitizeUpstreamErrorBody(shape clienterror.Shape) gin.HandlerFunc {
	return SanitizeUpstreamErrorBodyWithLimit(shape, defaultClientErrorBodyLimit)
}

// SanitizeUpstreamErrorBodyWithLimit 同 SanitizeUpstreamErrorBody，可指定缓冲上限（测试用）。
func SanitizeUpstreamErrorBodyWithLimit(shape clienterror.Shape, limitBytes int) gin.HandlerFunc {
	if limitBytes <= 0 {
		limitBytes = defaultClientErrorBodyLimit
	}
	return func(c *gin.Context) {
		if c == nil || c.Writer == nil {
			return
		}
		writer := &sanitizingWriter{ResponseWriter: c.Writer, ginCtx: c, shape: shape, limit: limitBytes}
		c.Writer = writer
		c.Next()
		writer.finalize()
	}
}

// upstreamErrorRecorded 判断本次请求是否真的碰到上游错误（而不是网关自己产生的错误）。
func upstreamErrorRecorded(c *gin.Context) bool {
	if c == nil {
		return false
	}
	for _, key := range []string{
		service.OpsUpstreamStatusCodeKey,
		service.OpsUpstreamErrorMessageKey,
		service.OpsUpstreamErrorDetailKey,
		service.OpsUpstreamErrorsKey,
	} {
		if v, ok := c.Get(key); ok && v != nil {
			return true
		}
	}
	return false
}

type sanitizingWriter struct {
	gin.ResponseWriter

	ginCtx *gin.Context
	shape  clienterror.Shape
	limit  int

	status      int
	buf         bytes.Buffer
	buffering   bool
	passthrough bool
	overflow    bool
	wrote       bool
}

func (w *sanitizingWriter) WriteHeader(code int) {
	if w.buffering {
		w.status = code
		return
	}
	if w.passthrough {
		return
	}
	contentType := w.Header().Get("Content-Type")
	if code < http.StatusBadRequest || isEventStream(contentType) || !isJSONContentType(contentType) {
		w.passthrough = true
		w.wrote = true
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.buffering = true
	w.status = code
	w.wrote = true
	// 刻意不调用底层 WriteHeader：等 finalize 决定是原样写出还是重建后写出。
}

func (w *sanitizingWriter) Write(b []byte) (int, error) {
	if w.passthrough {
		w.wrote = true
		return w.ResponseWriter.Write(b)
	}
	if !w.buffering {
		// 未声明状态就写：按 200 处理，直接直通。
		w.passthrough = true
		w.wrote = true
		return w.ResponseWriter.Write(b)
	}
	if w.buf.Len()+len(b) > w.limit {
		w.overflow = true
		return len(b), nil // 丢弃原始内容，finalize 时只发网关文案
	}
	return w.buf.Write(b)
}

func (w *sanitizingWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *sanitizingWriter) Written() bool { return w.wrote }

func (w *sanitizingWriter) Status() int {
	if w.buffering || w.passthrough {
		return w.status
	}
	return w.ResponseWriter.Status()
}

func (w *sanitizingWriter) Size() int {
	if w.buffering {
		return w.buf.Len()
	}
	return w.ResponseWriter.Size()
}

// Flush 用于流式输出：一旦要求 flush 就说明不能再缓冲，先按当前状态收口再直通。
func (w *sanitizingWriter) Flush() {
	if w.buffering {
		w.finalize()
		w.passthrough = true
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *sanitizingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.passthrough = true
	w.buffering = false
	return w.ResponseWriter.Hijack()
}

func (w *sanitizingWriter) CloseNotify() <-chan bool {
	return w.ResponseWriter.CloseNotify()
}

func (w *sanitizingWriter) finalize() {
	if w.passthrough || !w.buffering {
		return
	}
	w.buffering = false

	var body []byte
	// 缓冲溢出时原始内容已被丢弃，只能发网关文案；这与「宁可少说话也不泄漏」一致。
	if w.overflow || upstreamErrorRecorded(w.ginCtx) {
		var upstream []byte
		if !w.overflow {
			upstream = w.buf.Bytes()
		}
		body = clienterror.Sanitize(w.shape, w.status, upstream)
		header := w.Header()
		header.Del("Content-Length")
		header.Set("Content-Type", "application/json; charset=utf-8")
	} else {
		body = append([]byte(nil), w.buf.Bytes()...)
	}

	w.wrote = true
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = w.ResponseWriter.Write(body)
}

func isEventStream(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

func isJSONContentType(contentType string) bool {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.Contains(strings.ToLower(contentType), "json")
	}
	mediaType = strings.ToLower(mediaType)
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}
