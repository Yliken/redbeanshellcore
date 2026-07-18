package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

func TestErrorLogIncludesResponseStatus(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := Middleware(WithLogger(logger))(func(context.Context, *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.StatusCode = 500
		return resp, core.NewOpError(core.ErrRemoteRuntime, "info", "node-1", "remote failed", nil)
	})
	req := core.NewRequest("info")
	req.NodeID = "node-1"

	_, _ = handler(context.Background(), req)
	logLine := output.String()
	for _, want := range []string{`"level":"ERROR"`, `"status":500`, `"operation":"info"`} {
		if !strings.Contains(logLine, want) {
			t.Fatalf("日志缺少 %s: %s", want, logLine)
		}
	}
}
