// Funcs.java：headless 用。參數＝起 迄（hex）。列出範圍內函式：入口、大小、被呼叫次數、前 16 bytes。不輸出反編譯碼。
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.Function;

public class Funcs extends GhidraScript {
  @Override public void run() throws Exception {
    String[] a = getScriptArgs();
    for (Function f : currentProgram.getFunctionManager().getFunctions(toAddr(a[0]), true)) {
      if (f.getEntryPoint().compareTo(toAddr(a[1])) >= 0) break;
      StringBuilder sig = new StringBuilder();
      for (byte b : getBytes(f.getEntryPoint(), 16)) sig.append(String.format("%02x", b));
      println(String.format("FN\t%s\t%d\t%d\t%s", f.getEntryPoint(), f.getBody().getNumAddresses(),
        getReferencesTo(f.getEntryPoint()).length, sig));
    }
  }
}
