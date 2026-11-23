// Decomp.java：headless 用。參數＝輸出檔 函式位址…。反編譯結果寫到 repo 外（stage3：反編譯碼不入版控、不貼筆記）。
import ghidra.app.script.GhidraScript;
import ghidra.app.decompiler.*;
import ghidra.program.model.listing.Function;
import java.io.PrintWriter;

public class Decomp extends GhidraScript {
  @Override public void run() throws Exception {
    String[] a = getScriptArgs();
    DecompInterface di = new DecompInterface(); di.openProgram(currentProgram);
    try (PrintWriter w = new PrintWriter(a[0])) {
      for (int i = 1; i < a.length; i++) {
        Function f = getFunctionContaining(toAddr(a[i]));
        if (f == null) f = createFunction(toAddr(a[i]), null); // 未定義的（如 bsearch 比較器）就地建立，-readOnly 不存檔
        if (f == null) { w.println("// " + a[i] + " no function"); continue; }
        DecompileResults r = di.decompileFunction(f, 60, monitor);
        w.println("// ==== " + f.getEntryPoint() + " " + f.getName());
        w.println(r.decompileCompleted() ? r.getDecompiledFunction().getC() : "// failed: " + r.getErrorMessage());
      }
    }
  }
}
