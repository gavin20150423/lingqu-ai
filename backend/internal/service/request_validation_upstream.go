package service

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

const upstreamRequestValidationInspectLimit = 128 * 1024

var upstreamRequiredFieldPattern = regexp.MustCompile(`(?i)^field\s+[^\s]+\s+is\s+required(?:\s|\(|$)`)

// isUpstreamRequestValidationBody identifies the Claude upstream response
// shape used for malformed messages requests. Although the upstream returned
// HTTP 500, changing accounts cannot repair a request that is missing a field.
func isUpstreamRequestValidationBody(statusCode int, body []byte) bool {
	if statusCode < 500 || len(body) == 0 {
		return false
	}
	if strings.TrimSpace(gjson.GetBytes(body, "error.type").String()) != "new_api_error" {
		return false
	}
	message := strings.TrimSpace(gjson.GetBytes(body, "error.message").String())
	return upstreamRequiredFieldPattern.MatchString(message)
}

// isUpstreamRequestValidationResponse peeks at the body and restores it so
// retry/failover classification does not consume the response payload.
func isUpstreamRequestValidationResponse(resp *http.Response) bool {
	if resp == nil || resp.Body == nil || resp.StatusCode < 500 {
		return false
	}
	originalBody := resp.Body
	body, err := io.ReadAll(io.LimitReader(originalBody, upstreamRequestValidationInspectLimit))
	resp.Body = &replayReadCloser{
		Reader: io.MultiReader(bytes.NewReader(body), originalBody),
		closer: originalBody,
	}
	return err == nil && isUpstreamRequestValidationBody(resp.StatusCode, body)
}
