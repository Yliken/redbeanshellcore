package wire

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	ProtocolVersion = "1"
	HeaderDelim     = "BODY"
	ResponsePrefix  = "RBS1.0"
)

type RequestEnvelope struct {
	Version string
	RID     string
	TS      string
	Nonce   string
	Sig     string
}

type ResponseHeader struct {
	Version string
	RID     string
	TS      string
	Nonce   string
	Status  string
	Sig     string
}

func NewRequestEnvelope(rid string, payload string, key string) *RequestEnvelope {
	ts := fmt.Sprintf("%d", time.Now().UnixMilli())
	nonce := generateNonce()
	env := &RequestEnvelope{
		Version: ProtocolVersion,
		RID:     rid,
		TS:      ts,
		Nonce:   nonce,
		Sig:     "",
	}
	if key != "" {
		env.Sig = computeHMAC(env.SigningPayload(payload), key)
	}
	return env
}

func (e *RequestEnvelope) SigningPayload(payload string) string {
	return e.Version + "|" + e.RID + "|" + e.TS + "|" + e.Nonce + "|" + payload
}

func (e *RequestEnvelope) FormFields() map[string]string {
	return map[string]string{
		"_v":     e.Version,
		"_rid":   e.RID,
		"_ts":    e.TS,
		"_nonce": e.Nonce,
		"_sig":   e.Sig,
	}
}

func ParseResponse(body []byte) (content []byte, header *ResponseHeader) {
	s := string(body)
	idx := strings.Index(s, ResponsePrefix)
	if idx < 0 {
		return body, nil
	}
	headerSection := s[idx:]
	bodyIdx := strings.Index(headerSection, HeaderDelim)
	if bodyIdx < 0 {
		return body, nil
	}
	bodyContentStart := idx + bodyIdx + len(HeaderDelim)
	if bodyContentStart < len(s) && s[bodyContentStart] == '\n' {
		bodyContentStart++
	}
	headerLines := s[idx : idx+bodyIdx]
	header = &ResponseHeader{Version: ProtocolVersion}
	for _, line := range strings.Split(headerLines, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == ResponsePrefix || line == HeaderDelim {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}
		switch parts[0] {
		case "RID":
			header.RID = parts[1]
		case "TS":
			header.TS = parts[1]
		case "NONCE":
			header.Nonce = parts[1]
		case "STATUS":
			header.Status = parts[1]
		case "SIG":
			header.Sig = parts[1]
		}
	}
	bodyContent := s[bodyContentStart:]
	if sigIdx := strings.Index(bodyContent, "\nSIG="); sigIdx >= 0 {
		bodyContent = bodyContent[:sigIdx]
	}
	if bodyContentStart < len(s) {
		content = []byte(strings.TrimSpace(bodyContent))
	} else {
		content = []byte{}
	}
	return content, header
}

func VerifyResponseHMAC(body []byte, sig string, key string) bool {
	if key == "" || sig == "" {
		return true
	}
	expected := computeHMAC(string(body), key)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func computeHMAC(message, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateNonce() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
