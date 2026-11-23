// calc prints expected melee or single-target direct spell damage.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"

	"wiz8-mod/internal/calc"
)

const limitations = "未支援：遠程、AoE 名單、施法成功率、Kill 抗性、狀態效果、F01 精確 RNG；近戰不含失手換目標及行動中斷。"
const buildLimitations = "build 未支援：負重（E、e 取 0）、裝備屬性加成、特殊物品傷害分支（F28 10×Z）、練技成長、背刺、Kill 抗性。"
const usage = `用法：
  calc party -exe <Wiz8.exe> [-metric <attr:i,l100:j,…>]（不給 -metric：14 個單一指標各印一段前緣，段間空行）
  calc f <節點位址> <輸入…>
  calc melee <scenario.json>
  calc build -exe <Wiz8.exe> -dbs <dir> <scenario.json>
  calc rank -exe <Wiz8.exe> -dbs <dir> [-top N] [-j N] <template.json>
  calc fight -exe <Wiz8.exe> -dbs <dir> [-nodes] <scenario.json>（build 欄位去 Target、加 Monster、Difficulty；輸出 TSV）
  calc spell -dbs <dir> -id <n> -pl <1..7> -L <n> [-pc <skill>] -monster <id>
build JSON 欄位：
  Race, Class, Level, Prio:[屬性索引], Skills:{"技能 id":裸值},
  Main:{Item,Skill}, Off:{Item,Skill}|null, Mode, OffMode,
  Target:{DefHit,DefPen,Attr4Sum,DA}
Race、Class、Item、Skill 均為零起算；Item 是 ITEMS.DBS id。
屬性索引依序為 STR INT PIE VIT DEX SPD SEN；Mode、OffMode 為 0..8，分別指定兩手。
build 不支援：負重、裝備屬性加成、特殊物品傷害分支、練技成長、背刺、Kill 抗性。
melee JSON 欄位（區分 A=攻擊預算與 ARhit=命中值）：
  ARhit, ARpen, A, V, Dice:{Z,N,S}, G, BonusPct, DA, SlotSum, Skill, LSLevel, OffHand
G 必須包含 mode 3 等額外骰組；OffHand 預設 false，LSLevel=0 停用 LS。
兩手各執行一次 melee，再將揮擊數與傷害相加。法術 id、monster id 為零起算。
-pc 未指定時停用 Power Cast；-pc 0 表示啟用（b=1）。
`

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "calc:", err)
		os.Exit(2)
	}
}
func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(out, usage)
		fmt.Fprintln(out, limitations)
		return nil
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	switch args[0] {
	case "party":
		return runParty(args[1:], out)
	case "f":
		return runNode(args[1:], out)
	case "rank":
		return runRank(args[1:], out)
	case "fight":
		return runFight(args[1:], out)
	case "build":
		if err := runBuild(args[1:], w); err != nil {
			return err
		}
	case "melee":
		if len(args) != 2 {
			return errors.New(usage)
		}
		f, err := os.Open(args[1])
		if err != nil {
			return err
		}
		defer f.Close()
		decoder := json.NewDecoder(f)
		decoder.DisallowUnknownFields()
		var in calc.MeleeInput
		if err := decoder.Decode(&in); err != nil {
			return fmt.Errorf("scenario: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return errors.New("scenario 必須只有一個 JSON 物件")
		}
		if in.G < 1 || in.G > 255 {
			return errors.New("G 必須為 1..255（已含模式與狀態加成）")
		}
		if in.Skill < 0 || in.Skill > 125 || in.LSLevel < 0 {
			return errors.New("Skill 必須為 0..125，LSLevel 不可為負數")
		}
		r := calc.MeleeRound(in)
		fmt.Fprintln(w, "項目\t期望值")
		fmt.Fprintf(w, "每回合揮擊數\t%.6f\n有效揮擊數（命中且穿甲）\t%.6f\n每回合傷害\t%.6f\n每揮 Kill 門檻（抗性前，非成功率）\t%.6f\n失手換目標\t未支援\n", r.ExpectedSwings, r.ExpectedEffectiveSwings, r.ExpectedDamage, r.KillThresholdPerSwingBeforeResistance)
	case "spell":
		fs := flag.NewFlagSet("spell", flag.ContinueOnError)
		fs.SetOutput(out)
		dbs := fs.String("dbs", os.Getenv("WIZ8_DBS"), "DATABASES 目錄")
		id := fs.Int("id", -1, "法術 id（零起算）")
		pl := fs.Int("pl", 1, "power level 1..7")
		level := fs.Int("L", 1, "有效施法等級")
		pc := fs.Int("pc", -1, "Power Cast 技能，省略則停用")
		monster := fs.Int("monster", -1, "怪物 id（零起算）")
		if err := fs.Parse(args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		if fs.NArg() != 0 || *dbs == "" || *id < 0 || *monster < 0 || *pl < 1 || *pl > 7 || *level < 1 || *pc < -1 || *pc > 125 {
			return errors.New("spell 參數無效\n" + usage)
		}
		spells, err := calc.LoadSpells(filepath.Join(*dbs, "SPELLTABLES.DBS"))
		if err != nil {
			return err
		}
		monsters, err := calc.LoadMonsters(filepath.Join(*dbs, "MONSTERS.DBS"))
		if err != nil {
			return err
		}
		if *id >= len(spells) || *monster >= len(monsters) {
			return errors.New("法術或怪物 id 超出資料範圍")
		}
		sp, target := spells[*id], monsters[*monster]
		b := calc.PowerCastB(max(0, *pc), *pc >= 0)
		r := calc.SpellCastID(sp, *id, *level, *pl, b, target)
		if r.UnsupportedReason != "" {
			return errors.New(r.UnsupportedReason)
		}
		fmt.Fprintln(w, "項目\t數值")
		fmt.Fprintf(w, "法術\t%d %s\n目標（模板抗性）\t%d %s\nPL\t%d\nPower Cast b\t%d\n單目標期望傷害\t%.6f\nSP 成本\t%d\n強度 S\t%d\n", *id, sp.Name, *monster, target.Name, *pl, b, r.ExpectedDamage, r.Cost, r.Strength)
	default:
		return errors.New(usage)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, limitations)
	return err
}

type buildWeapon struct {
	Item, Skill int
}

type buildScenario struct {
	Race, Class, Level int
	Prio               []int
	Skills             map[int]int
	Main, Off          *buildWeapon
	Mode, OffMode      int
	Target             calc.Target
}

func readBuildScenario(r io.Reader) (buildScenario, error) {
	var in *buildScenario
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(&in); err != nil {
		return buildScenario{}, fmt.Errorf("scenario: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return buildScenario{}, errors.New("scenario 必須只有一個 JSON 物件")
	}
	if in == nil || in.Main == nil {
		return buildScenario{}, errors.New("build scenario 必須有 Main 武器")
	}
	if in.Mode < 0 || in.Mode > 8 || in.OffMode < 0 || in.OffMode > 8 {
		return buildScenario{}, errors.New("Mode、OffMode 必須為 0..8")
	}
	for s, v := range in.Skills {
		if s < 0 || s >= 41 {
			return buildScenario{}, errors.New("Skills id 必須為 0..40")
		}
		if v < 0 || v > 125 {
			return buildScenario{}, errors.New("Skills 值必須為 0..125")
		}
	}
	return *in, nil
}

func runBuild(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(out)
	exe := fs.String("exe", os.Getenv("WIZ8_EXE"), "Wiz8.exe 路徑（只讀）")
	dbs := fs.String("dbs", os.Getenv("WIZ8_DBS"), "DATABASES 目錄（只讀）")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(out, usage)
			return nil
		}
		return err
	}
	if fs.NArg() != 1 || *exe == "" || *dbs == "" {
		return errors.New("build 參數無效\n" + usage)
	}
	f, err := os.Open(fs.Arg(0))
	if err != nil {
		return err
	}
	defer f.Close()
	in, err := readBuildScenario(f)
	if err != nil {
		return err
	}
	x, err := calc.LoadExe(*exe)
	if err != nil {
		return err
	}
	attrs, err := calc.PlanAttrs(x, in.Race, in.Class, in.Level, in.Prio)
	if err != nil {
		return err
	}
	items, err := calc.LoadItems(filepath.Join(*dbs, "ITEMS.DBS"))
	if err != nil {
		return err
	}
	weapon := func(w *buildWeapon) (calc.Weapon, error) {
		if w.Item < 0 || w.Item >= len(items) || w.Skill < 0 || w.Skill >= 41 {
			return calc.Weapon{}, errors.New("武器 Item 或 Skill 超出資料範圍")
		}
		return calc.Weapon{Item: items[w.Item], Skill: w.Skill}, nil
	}
	main, err := weapon(in.Main)
	if err != nil {
		return err
	}
	var off *calc.Weapon
	if in.Off != nil {
		w, err := weapon(in.Off)
		if err != nil {
			return err
		}
		off = &w
	}
	b := calc.Build{Class: in.Class, Level: in.Level, Attr: attrs}
	for s, raw := range in.Skills {
		b.Skill[s] = raw
	}
	r := calc.MeleeBuild(x, b, main, off, in.Mode, in.OffMode, in.Target)
	fmt.Fprintln(out, "屬性\tSTR\tINT\tPIE\tVIT\tDEX\tSPD\tSEN")
	fmt.Fprintf(out, "最終裸值\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", attrs[0], attrs[1], attrs[2], attrs[3], attrs[4], attrs[5], attrs[6])
	fmt.Fprintln(out, "手部\tItem\t武器\t模式\tK\tM\tA\tV\tBARraw\tHitMod\tDmgPct\tARhit\tARpen\t期望揮擊數\t有效揮擊數\t期望傷害")
	printHand := func(name string, id int, w calc.Weapon, mode int, h calc.HandCache, hit, pen int, result calc.MeleeResult) {
		fmt.Fprintf(out, "%s\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%.6f\t%.6f\t%.6f\n", name, id, w.Item.Name, mode, h.K, h.M, h.A, h.V, h.BARraw, h.HitMod, h.DmgPct, hit, pen, result.ExpectedSwings, result.ExpectedEffectiveSwings, result.ExpectedDamage)
	}
	printHand("主手", in.Main.Item, main, in.Mode, r.Main, r.MainHit, r.MainPen, r.MainRound)
	swings := r.MainRound.ExpectedSwings
	if off != nil {
		printHand("副手", in.Off.Item, *off, in.OffMode, r.Off, r.OffHit, r.OffPen, *r.OffRound)
		swings += r.OffRound.ExpectedSwings
	}
	fmt.Fprintf(out, "合計期望揮擊數\t%.6f\n合計每回合期望傷害\t%.6f\n", swings, r.ExpectedDamage)
	_, err = fmt.Fprintln(out, buildLimitations)
	return err
}

func runNode(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("f 需要節點位址")
	}
	node, ok := calc.Nodes[args[0]]
	if !ok {
		return fmt.Errorf("未知節點 %s", args[0])
	}
	if len(args)-1 != len(node.In) {
		return fmt.Errorf("%s 輸入：%s", node.Addr, strings.Join(node.In, " "))
	}
	in := make([]int64, len(node.In))
	for i, arg := range args[1:] {
		v, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return fmt.Errorf("%s: %w", node.In[i], err)
		}
		in[i] = v
	}
	dist, err := calc.NodeDist(node, in)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "dist")
	mean := make([]float64, len(node.Out))
	for _, row := range dist {
		values := make([]string, len(row.Out))
		for i, v := range row.Out {
			values[i] = strconv.FormatInt(v, 10)
			mean[i] += float64(v) * row.Probability
		}
		fmt.Fprintf(out, "%s: %.12g\n", strings.Join(values, ","), row.Probability)
	}
	for i, name := range node.Out {
		fmt.Fprintf(out, "mean %s: %.12g\n", name, mean[i])
	}
	fmt.Fprintf(out, "in: %s\nout: %s\n", strings.Join(node.In, ", "), strings.Join(node.Out, ", "))
	return nil
}

func runParty(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("party", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	exe := fs.String("exe", "", "Wiz8.exe 路徑（只讀）")
	metric := fs.String("metric", "", "attr:i 或 l100:i，以逗號分隔")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err = fmt.Fprint(out, usage)
			return err
		}
		return err
	}
	if fs.NArg() != 0 || *exe == "" {
		return errors.New("party 必須指定 -exe <Wiz8.exe>，且不接受位置參數")
	}
	names := []string{*metric}
	if !isFlagSet(fs, "metric") { // goals/stage10.md B3：不指定指標就輸出各指標的前緣
		names = nil
		for _, k := range []string{"attr", "l100"} {
			for i := 0; i < 7; i++ {
				names = append(names, fmt.Sprintf("%s:%d", k, i))
			}
		}
	}
	var all [][]calc.PartyMetric
	for _, n := range names {
		m, err := calc.ParsePartyMetrics(n)
		if err != nil {
			return err
		}
		all = append(all, m)
	}
	tables, err := calc.LoadExe(*exe)
	if err != nil {
		return err
	}
	var buf strings.Builder
	for i, m := range all {
		if i > 0 {
			buf.WriteString("\n")
		}
		if err := writeParty(&buf, tables, m); err != nil {
			return err
		}
	}
	_, err = io.WriteString(out, buf.String())
	return err
}

func writeParty(out io.Writer, tables *calc.ExeTables, metrics []calc.PartyMetric) error {
	points, err := calc.Party(tables, metrics)
	if err != nil {
		return err
	}
	header := []string{"race", "class", "gender", "attrs"}
	for _, m := range metrics {
		header = append(header, m.Name)
	}
	if _, err := fmt.Fprintln(out, strings.Join(header, "\t")); err != nil {
		return err
	}
	for _, p := range points {
		var attrs [7]string
		for i, a := range p.Attrs {
			attrs[i] = strconv.Itoa(a)
		}
		row := []string{strconv.Itoa(p.Race), strconv.Itoa(p.Class), strconv.Itoa(p.Gender), strings.Join(attrs[:], ",")}
		for _, m := range metrics {
			row = append(row, strconv.Itoa(m.Value(p.Attrs)))
		}
		if _, err := fmt.Fprintln(out, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return nil
}

func isFlagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) { set = set || f.Name == name })
	return set
}

func runFight(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("fight", flag.ContinueOnError)
	fs.SetOutput(out)
	exe := fs.String("exe", os.Getenv("WIZ8_EXE"), "Wiz8.exe 路徑（只讀）")
	dbs := fs.String("dbs", os.Getenv("WIZ8_DBS"), "DATABASES 目錄（只讀）")
	nodes := fs.Bool("nodes", false, "只列出用到的節點位址")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(out, usage)
			return nil
		}
		return err
	}
	if fs.NArg() != 1 || *exe == "" || *dbs == "" {
		return errors.New("fight 參數無效\n" + usage)
	}
	f, err := os.Open(fs.Arg(0))
	if err != nil {
		return err
	}
	defer f.Close()
	var in *calc.FightScenario
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err := d.Decode(&in); err != nil {
		return fmt.Errorf("scenario: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("scenario 必須只有一個 JSON 物件")
	}
	if in == nil {
		return errors.New("fight scenario 必須是 JSON 物件")
	}
	if in.Main == nil && in.Off != nil {
		return errors.New("fight scenario 空手 Main 不支援 Off 武器")
	}
	if in.Mode < 0 || in.Mode > 8 || in.OffMode < 0 || in.OffMode > 8 {
		return errors.New("Mode、OffMode 必須為 0..8")
	}
	if *nodes {
		for _, a := range calc.FightNodes {
			fmt.Fprintln(out, a)
		}
		return nil
	}
	w, err := calc.LoadFightWorld(*exe, *dbs)
	if err != nil {
		return err
	}
	kv, err := w.Fight(*in, nil)
	if err != nil {
		return err
	}
	for _, e := range kv {
		fmt.Fprintf(out, "%s\t%.6f\n", e.Key, e.Val)
	}
	return nil
}

func runRank(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("rank", flag.ContinueOnError)
	fs.SetOutput(out)
	exe := fs.String("exe", os.Getenv("WIZ8_EXE"), "Wiz8.exe 路徑（只讀）")
	dbs := fs.String("dbs", os.Getenv("WIZ8_DBS"), "DATABASES 目錄（只讀）")
	workers := fs.Int("j", runtime.GOMAXPROCS(0), "rank worker 數")
	top := fs.Int("top", 0, "依最後等級傷害印前 N 列；0 印全部")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 || *exe == "" || *dbs == "" || *top < 0 || *workers < 1 {
		return errors.New("rank 參數無效\n" + usage)
	}
	f, err := os.Open(fs.Arg(0))
	if err != nil {
		return err
	}
	defer f.Close()
	t, err := calc.ReadRankTemplate(f)
	if err != nil {
		return err
	}
	w, err := calc.LoadFightWorld(*exe, *dbs)
	if err != nil {
		return err
	}
	return calc.WriteRankWorkers(out, w, t, *top, *workers)
}
