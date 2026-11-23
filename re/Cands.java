// Cands.java：headless 用。每個函式一列：入口、大小、前 16 bytes、直接呼叫的函式、引用的記錄基址（參數 1＝全域位址清單，hex，逗號分隔）、
// 所有資料引用（D1 用）、基本塊起點（D4 覆蓋用）。
// 輸出 TSV 到參數 2 的路徑（repo 外，原版衍生）。不輸出反編譯碼。候選判定在 scripts/formula-candidates.sh。
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.*;
import ghidra.program.model.address.*;
import ghidra.program.model.symbol.*;
import java.io.PrintWriter;
import java.util.*;
import ghidra.program.model.block.*;

public class Cands extends GhidraScript {
  @Override public void run() throws Exception {
    String[] a = getScriptArgs();
    Set<Long> want = new HashSet<>();
    for (String s : a[0].split(",")) want.add(Long.parseLong(s.replace("0x", ""), 16));
    try (PrintWriter w = new PrintWriter(a[1])) {
      Listing L = currentProgram.getListing();
      for (Function f : currentProgram.getFunctionManager().getFunctions(true)) {
        StringBuilder sig = new StringBuilder();
        for (byte b : getBytes(f.getEntryPoint(), 16)) sig.append(String.format("%02x", b));
        TreeSet<String> calls = new TreeSet<>(), refs = new TreeSet<>(), data = new TreeSet<>(), blocks = new TreeSet<>();
        for (Instruction i : L.getInstructions(f.getBody(), true)) {
          for (Reference r : i.getReferencesFrom()) {
            long t = r.getToAddress().getOffset();
            if (r.getReferenceType().isCall()) calls.add(String.format("%x", t));
            else if (r.getReferenceType().isData()) {
              data.add(String.format("%x", t));
              if (want.contains(t)) refs.add(String.format("%x", t));
            }
          }
        }
        CodeBlockIterator bi = new BasicBlockModel(currentProgram).getCodeBlocksContaining(f.getBody(), monitor);
        while (bi.hasNext()) blocks.add(String.format("%x", bi.next().getFirstStartAddress().getOffset()));
        w.printf("%x\t%d\t%s\t%s\t%s\t%s\t%s%n", f.getEntryPoint().getOffset(), f.getBody().getNumAddresses(), sig,
          String.join(",", calls), String.join(",", refs), String.join(",", data), String.join(",", blocks));
      }
    }
  }
}
