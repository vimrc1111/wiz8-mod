// Xrefs.java：headless 用。參數＝要找的字串（完全相等），或 @位址（列出引用）。每條印：字串位址、引用位址、所在函式入口、函式前 16 bytes。
// 輸出只有位址與 bytes，不含反編譯碼（stage3 邊界）。
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.*;
import ghidra.program.model.symbol.Reference;


public class Xrefs extends GhidraScript {
  @Override public void run() throws Exception {
    for (String want : getScriptArgs()) {
      if (want.startsWith("@")) { // @位址：列出對該位址的引用
        for (Reference r : getReferencesTo(toAddr(want.substring(1)))) {
          Function f = getFunctionContaining(r.getFromAddress());
          println(String.format("XREF\t%s\t%s\t%s\t%s", want, r.getFromAddress(), r.getReferenceType(), f == null ? "-" : f.getEntryPoint().toString()));
        }
        continue;
      }
      boolean hit = false;
      for (Data d : currentProgram.getListing().getDefinedData(true)) {
        if (!d.hasStringValue()) continue;
        Object v = d.getValue();
        if (v == null || !want.equals(v.toString())) continue;
        hit = true;
        for (Reference r : getReferencesTo(d.getAddress())) {
          Function f = getFunctionContaining(r.getFromAddress());
          String fe = f == null ? "-" : f.getEntryPoint().toString();
          StringBuilder sig = new StringBuilder();
          if (f != null) for (byte b : getBytes(f.getEntryPoint(), 16)) sig.append(String.format("%02x", b));
          println(String.format("XREF\t%s\t%s\t%s\t%s\t%s", want, d.getAddress(), r.getFromAddress(), fe, sig));
        }
      }
      if (!hit) println("XREF\t" + want + "\tNOT-FOUND");
    }
  }
}
