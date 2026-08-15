package service

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

const upstreamContentPolicyInspectLimit = 128 * 1024

type replayReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r *replayReadCloser) Close() error {
	return r.closer.Close()
}

const upstreamContentPolicyClientMessage = "Request blocked by upstream content policy"

// isUpstreamContentPolicyBody identifies request-scoped safety refusals. They
// must not trigger account retries or failover because changing credentials
// cannot change the rejected prompt. Keep this narrow so account entitlement
// and permission failures retain their normal failover behavior.
func isUpstreamContentPolicyBody(statusCode int, body []byte) bool {
	if statusCode != http.StatusForbidden || len(body) == 0 {
		return false
	}
	lower := strings.ToLower(string(body))
	for _, marker := range []string{
		"flagged by the content review system",
		"blocked by the content review system",
		"blocked according to the usage policy",
		"request blocked by content policy",
		"request blocked by upstream content policy",
		"request rejected by content policy",
		"content policy violation",
		"content_policy_violation",
		"content_policy_error",
		"content moderation blocked",
		"content moderation rejected",
		"request violates content policy",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// isUpstreamContentPolicyResponse peeks at a 403 body and restores it so the
// existing error handling and logging paths can consume the response normally.
func isUpstreamContentPolicyResponse(resp *http.Response) bool {
	if resp == nil || resp.StatusCode != http.StatusForbidden || resp.Body == nil {
		return false
	}
	originalBody := resp.Body
	body, err := io.ReadAll(io.LimitReader(originalBody, upstreamContentPolicyInspectLimit))
	resp.Body = &replayReadCloser{
		Reader: io.MultiReader(bytes.NewReader(body), originalBody),
		closer: originalBody,
	}
	return err == nil && isUpstreamContentPolicyBody(resp.StatusCode, body)
}
