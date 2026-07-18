package audit

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

type recordingSink struct {
	events []AuditEvent
}

func (s *recordingSink) Record(event AuditEvent) error {
	s.events = append(s.events, event)
	return nil
}

func testRequest() *core.Request {
	req := core.NewRequest("info")
	req.ID = "request-1"
	req.NodeID = "node-1"
	return req
}

func TestWithLoggerConfiguresDefaultSink(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := Middleware(WithLogger(logger))(func(context.Context, *core.Request) (*core.Response, error) {
		return core.NewResponse(), nil
	})

	if _, err := handler(context.Background(), testRequest()); err != nil {
		t.Fatalf("handler 出错: %v", err)
	}
	if !strings.Contains(output.String(), `"msg":"audit"`) {
		t.Fatalf("自定义 logger 未收到默认 sink 输出: %s", output.String())
	}
}

func TestWithNilSinkFallsBackSafely(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := Middleware(WithSink(nil), WithLogger(logger))(func(context.Context, *core.Request) (*core.Response, error) {
		return core.NewResponse(), nil
	})

	if _, err := handler(context.Background(), testRequest()); err != nil {
		t.Fatalf("handler 出错: %v", err)
	}
	if output.Len() == 0 {
		t.Fatal("nil sink 应回退到默认 sink")
	}
}

func TestAuditRecordsFinalErrorKind(t *testing.T) {
	sink := &recordingSink{}
	sentinel := core.NewOpError(core.ErrParse, "info", "node-1", "parse failed", errors.New("boom"))
	handler := Middleware(WithSink(sink))(func(context.Context, *core.Request) (*core.Response, error) {
		return core.NewResponse(), sentinel
	})

	_, _ = handler(context.Background(), testRequest())
	if len(sink.events) != 1 {
		t.Fatalf("期望 1 条事件，got %d", len(sink.events))
	}
	event := sink.events[0]
	if event.Success || event.ErrorKind != string(core.ErrParse) {
		t.Fatalf("审计错误分类不正确: %+v", event)
	}
}
