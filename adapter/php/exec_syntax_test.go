package php

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecTemplatePHP7Syntax guards the generated exec template against
// regressions like missing semicolons before closing braces.
func TestExecTemplatePHP7Syntax(t *testing.T) {
	phpBin, err := exec.LookPath("php")
	if err != nil {
		t.Skip("php not found")
	}
	op := NewPhpExec("id")
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "exec.php")
	src := "<?php\n" + string(req.Payload) + "\n?>"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(phpBin, "-l", path).CombinedOutput()
	if err != nil {
		t.Fatalf("php -l failed:\n%s\ncode:\n%s", out, src)
	}
	if !strings.Contains(string(out), "No syntax errors") {
		t.Fatalf("unexpected php -l output: %s", out)
	}
}
