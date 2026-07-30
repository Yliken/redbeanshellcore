package httpform

import (
	"net"
	"strconv"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

// ParseTransportOptions extracts transport options from a NodeRecord.
// Returns the configured options and whether wire protocol is enabled.
func ParseTransportOptions(rec *core.NodeRecord) (Options, bool) {
	opts := DefaultOptions()
	opts.Timeout = 30 * time.Second
	wireProto := false

	if rec.Config.Options == nil { return opts, false }

	if v, ok := rec.Config.Options["insecure_tls"]; ok && v == "true" {
		opts.InsecureTLS = true
	}
	if v, ok := rec.Config.Options["timeout"]; ok {
		if d, err := time.ParseDuration(v); err == nil { opts.Timeout = d }
	}
	if v, ok := rec.Config.Options["ua_rotation"]; ok && v == "true" {
		opts.UARotation = true; opts.UAPool = nil
	}
	if v, ok := rec.Config.Options["dynamic_fields"]; ok && v == "true" {
		opts.DynamicFieldNames = true; opts.FieldGen = NewFieldGenerator()
	}
	if v, ok := rec.Config.Options["honeypot_count"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 { opts.EnablePadding = true; opts.HoneypotCount = n }
	}
	if v, ok := rec.Config.Options["proxy"]; ok && v != "" {
		if host, portStr, err := net.SplitHostPort(v); err == nil {
			if port, portErr := strconv.Atoi(portStr); portErr == nil {
				opts.ProxyChain = []ProxyConfig{{Type: ProxyHTTP, Host: host, Port: port}}
			}
		}
	}
	if v, ok := rec.Config.Options["tls_fingerprint"]; ok && v == "true" {
		opts.TLSFingerprint.Enabled = true
	}
	if v, ok := rec.Config.Options["http_protocol"]; ok {
		switch v {
		case "http1.1": opts.Protocol = ProtocolHTTP11
		case "http2": opts.Protocol = ProtocolHTTP2
		case "http3": opts.Protocol = ProtocolHTTP3
		}
	}
	if v, ok := rec.Config.Options["max_idle_conns"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 { opts.MaxIdleConns = n }
	}
	if v, ok := rec.Config.Options["max_idle_per_host"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 { opts.MaxIdleConnsPerHost = n }
	}
	if v, ok := rec.Config.Options["cookie_jar"]; ok && v == "false" {
		opts.EnableCookieJar = false
	}
	if v, ok := rec.Config.Options["wire_protocol"]; ok && v == "true" {
		opts.WireProtocol = true; wireProto = true
	}

	return opts, wireProto
}
