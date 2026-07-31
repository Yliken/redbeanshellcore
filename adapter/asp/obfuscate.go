package asp

import (
	"crypto/rand"
	"encoding/hex"
)

type Obfuscator struct {
	paramZ1      string
	paramZ2      string
	helperDecode string
	helperEncode string
	helperBTS    string
	errPrefix    string
}

func NewObfuscator() *Obfuscator {
	return &Obfuscator{
		paramZ1:      randVar(8),
		paramZ2:      randVar(8),
		helperDecode: randVar(6),
		helperEncode: randVar(6),
		helperBTS:    randVar(8),
		errPrefix:    "ERR:" + randHex(8) + ":",
	}
}

func DefaultObfuscator() *Obfuscator {
	return &Obfuscator{
		paramZ1:      "z1",
		paramZ2:      "z2",
		helperDecode: "b64d",
		helperEncode: "b64e",
		helperBTS:    "BytesToStr",
		errPrefix:    "ERR:",
	}
}

func (o *Obfuscator) Param1() string     { return o.paramZ1 }
func (o *Obfuscator) Param2() string     { return o.paramZ2 }
func (o *Obfuscator) HelperDecode() string { return o.helperDecode }
func (o *Obfuscator) HelperEncode() string { return o.helperEncode }
func (o *Obfuscator) HelperBTS() string   { return o.helperBTS }
func (o *Obfuscator) ErrPrefix() string   { return o.errPrefix }

func (o *Obfuscator) HelperCode() string {
	h := o.HelperDecode()
	he := o.HelperEncode()
	bts := o.HelperBTS()
	return "Function " + h + "(s):With CreateObject(\"MSXML2.DOMDocument.6.0\").CreateElement(\"b64\"):.DataType=\"bin.base64\":.Text=s:" + h + "=" + bts + "(.NodeTypedValue):End With:End Function:" +
		"Function " + he + "(s):With CreateObject(\"MSXML2.DOMDocument.6.0\").CreateElement(\"b64\"):.DataType=\"bin.base64\":.Text=s:" + he + "=.Text:End With:End Function:" +
		"Function " + bts + "(b):Dim i,s:For i=1 To LenB(b):s=s&Chr(AscB(MidB(b,i,1))):Next:" + bts + "=s:End Function:"
}

func randVar(n int) string {
	buf := make([]byte, (n+1)/2)
	_, _ = rand.Read(buf)
	out := hex.EncodeToString(buf)
	if len(out) > n { out = out[:n] }
	if out[0] >= '0' && out[0] <= '9' { out = "x" + out }
	if len(out) > n { out = out[:n] }
	return out
}

func randHex(n int) string {
	buf := make([]byte, (n+1)/2)
	_, _ = rand.Read(buf)
	out := hex.EncodeToString(buf)
	if len(out) > n { out = out[:n] }
	return out
}
