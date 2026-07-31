package aspx

import (
	"crypto/rand"
	"encoding/hex"
)

type Obfuscator struct {
	paramZ1      string
	paramZ2      string
	helperDecode string
	helperEncode string
	className    string
	errPrefix    string
}

func NewObfuscator() *Obfuscator {
	return &Obfuscator{
		paramZ1:      randVar(8),
		paramZ2:      randVar(8),
		helperDecode: randVar(6),
		helperEncode: randVar(6),
		className:    "E" + randVar(6),
		errPrefix:    "ERR:" + randHex(8) + ":",
	}
}

func DefaultObfuscator() *Obfuscator {
	return &Obfuscator{
		paramZ1:      "z1",
		paramZ2:      "z2",
		helperDecode: "d",
		helperEncode: "e",
		className:    "E",
		errPrefix:    "ERR:",
	}
}

func (o *Obfuscator) Param1() string      { return o.paramZ1 }
func (o *Obfuscator) Param2() string      { return o.paramZ2 }
func (o *Obfuscator) HelperDecode() string { return o.helperDecode }
func (o *Obfuscator) HelperEncode() string { return o.helperEncode }
func (o *Obfuscator) ClassName() string    { return o.className }
func (o *Obfuscator) ErrPrefix() string    { return o.errPrefix }

func (o *Obfuscator) CsWrap(code string) string {
	dn := o.HelperDecode()
	en := o.HelperEncode()
	cn := o.ClassName()
	return "using System;using System.IO;using System.Text;using System.Diagnostics;" +
		"public class " + cn + "{" +
		"static string " + dn + "(string s){return Encoding.UTF8.GetString(Convert.FromBase64String(s));}" +
		"static string " + en + "(string s){return Convert.ToBase64String(Encoding.UTF8.GetBytes(s));}" +
		"public static void Run(){" + code + "}}"
}

func randVar(n int) string {
	buf := make([]byte, (n+1)/2)
	_, _ = rand.Read(buf)
	out := hex.EncodeToString(buf)
	if len(out) > n { out = out[:n] }
	if out[0] >= '0' && out[0] <= '9' { out = "x" + out }
	if len(out) > n { out = out[:n] }
	return "m" + out
}

func randHex(n int) string {
	buf := make([]byte, (n+1)/2)
	_, _ = rand.Read(buf)
	out := hex.EncodeToString(buf)
	if len(out) > n { out = out[:n] }
	return out
}
