package php

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestExecTemplatePHP7Syntax guards the generated exec template against
// regressions like missing semicolons or uninitialized function refs.
func TestExecTemplatePHP7Syntax(t *testing.T) {
	phpBin, err := exec.LookPath("php")
	if err != nil {
		t.Skip("php not found")
	}
	op := NewPhpExec("echo WST_EXEC_OK")
	if runtime.GOOS == "windows" {
		op = op.WithBin("cmd.exe")
	}
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	payloadPath := filepath.Join(dir, "payload.txt")
	if err := os.WriteFile(payloadPath, req.Payload, 0o600); err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	sb.WriteString("<?php\n")
	for k, v := range req.Params {
		sb.WriteString("$_POST['")
		sb.WriteString(k)
		sb.WriteString("']='")
		sb.WriteString(strings.ReplaceAll(string(v), "'", "\\'"))
		sb.WriteString("';\n")
	}
	sb.WriteString("$__rbs_code=file_get_contents('")
	sb.WriteString(strings.ReplaceAll(payloadPath, "\\", "/"))
	sb.WriteString("');eval($__rbs_code);\n")
	runnerPath := filepath.Join(dir, "run.php")
	if err := os.WriteFile(runnerPath, []byte(sb.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(phpBin, runnerPath).CombinedOutput()
	if err != nil {
		t.Fatalf("php run failed: %v\nout:\n%s", err, out)
	}
	if !strings.Contains(string(out), "WST_EXEC_OK") {
		t.Fatalf("exec output missing marker: %q", out)
	}
}
