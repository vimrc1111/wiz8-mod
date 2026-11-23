package calc

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// RankTemplate describes a single-class opening and its evaluation levels.
type RankTemplate struct {
	Attrs, Levels, SkillPrio           []int
	SkillSet                           []int
	SkillPrios                         [][]int
	Train                              map[int]int
	AttrMin                            int
	Haste                              bool
	Main, Off                          *FightWeapon
	Loadouts                           []RankLoadout
	Monsters                           []int
	Metric                             string
	Commands                           []string
	Mode, OffMode, Monster, Difficulty int
	Races, Classes                     []int
}

type RankLoadout struct {
	Name      string
	Main, Off *FightWeapon
}

type RankRow struct {
	Loadout, Note, Command string
	Race, Class            int
	Prio                   []int
	SkillPrio              []int
	Damage                 []float64 // In template Levels order; negative infinity represents NA.
}

// ReadRankTemplate rejects unknown fields and trailing JSON values.
func ReadRankTemplate(r io.Reader) (RankTemplate, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return RankTemplate{}, err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var t *RankTemplate
	if err := d.Decode(&t); err != nil {
		return RankTemplate{}, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return RankTemplate{}, fmt.Errorf("rank: expected one JSON object")
	}
	if t == nil {
		return RankTemplate{}, fmt.Errorf("rank: expected an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return RankTemplate{}, err
	}
	sources := 0
	null := false
	for field, v := range fields {
		if strings.EqualFold(field, "SkillPrio") || strings.EqualFold(field, "SkillSet") || strings.EqualFold(field, "SkillPrios") {
			sources++
			null = null || string(bytes.TrimSpace(v)) == "null"
		}
	}
	if sources != 1 || null {
		return RankTemplate{}, fmt.Errorf("rank: give exactly one skill priority source")
	}
	if _, err := rankSkillPrios(*t); err != nil {
		return RankTemplate{}, err
	}
	if t.Loadouts != nil {
		for field := range fields {
			if strings.EqualFold(field, "Main") || strings.EqualFold(field, "Off") {
				return RankTemplate{}, fmt.Errorf("rank: Loadouts cannot be combined with Main/Off")
			}
		}
	}
	hasCommands, hasMode := false, false
	for field := range fields {
		hasCommands = hasCommands || strings.EqualFold(field, "Commands")
		hasMode = hasMode || strings.EqualFold(field, "Mode") || strings.EqualFold(field, "OffMode")
	}
	if hasCommands && hasMode {
		return RankTemplate{}, fmt.Errorf("rank: Commands cannot be combined with Mode/OffMode")
	}
	if hasCommands && t.Commands == nil {
		t.Commands = []string{"attack", "berserk"}
	}
	if _, err := rankCommands(*t); err != nil {
		return RankTemplate{}, err
	}
	if err := rankMetricOptions(*t); err != nil {
		return RankTemplate{}, err
	}
	return *t, nil
}

func rankMetricOptions(t RankTemplate) error {
	if t.Metric != "" && t.Metric != "damage" && t.Metric != "kill" && t.Metric != "kill-hi" && t.Metric != "kill-swing" {
		return fmt.Errorf("rank: unknown metric %q", t.Metric)
	}
	if len(t.Monsters) > 0 && t.Monster != 0 {
		return fmt.Errorf("rank: Monsters cannot be combined with nonzero Monster")
	}
	return nil
}

func rankIDs(ids []int, n int, name string) ([]int, error) {
	if ids == nil {
		ids = make([]int, n)
		for i := range ids {
			ids[i] = i
		}
	}
	seen := make(map[int]bool)
	for _, id := range ids {
		if id < 0 || id >= n || seen[id] {
			return nil, fmt.Errorf("rank: invalid or duplicate %s id %d", name, id)
		}
		seen[id] = true
	}
	return ids, nil
}

// Rank enumerates race, class and every attribute-priority prefix permutation.
func Rank(w *FightWorld, t RankTemplate) ([]RankRow, error) {
	rows, _, err := rank(w, t)
	return rows, err
}

func rank(w *FightWorld, t RankTemplate) ([]RankRow, map[string]int, error) {
	rows, skipped, _, _, err := rankWorkers(w, t, runtime.GOMAXPROCS(0))
	return rows, skipped, err
}

func rankWorkers(w *FightWorld, t RankTemplate, workers int) ([]RankRow, map[string]int, int, int, error) {
	if workers < 1 {
		return nil, nil, 0, 0, fmt.Errorf("rank: workers must be positive")
	}
	rows, skipped, jobs, cells, err := rankPlan(w, t)
	if err != nil {
		return nil, skipped, 0, cells, err
	}
	queue := make(chan int)
	var wg sync.WaitGroup
	for i := 0; i < min(workers, len(jobs)); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range queue {
				job := &jobs[i]
				job.damage, job.err = rankDamage(w, job)
			}
		}()
	}
	for i := range jobs {
		queue <- i
	}
	close(queue)
	wg.Wait()
	fights := 0
	for _, job := range jobs {
		fights += job.fights
		if job.err != nil {
			return nil, skipped, len(jobs), cells, fmt.Errorf("rank race %d class %d level %d: %w", job.scenario.Race, job.scenario.Class, job.scenario.Level, job.err)
		}
		for _, dst := range job.dest {
			rows[dst[0]].Damage[dst[1]] = job.damage
		}
	}
	return rows, skipped, fights, cells, nil
}

type rankJob struct {
	scenario FightScenario
	command  string
	monsters []int
	metric   string
	dest     [][2]int
	damage   float64
	fights   int
	err      error
}

type rankKey struct {
	command                            string
	monsters, metric                   string
	race, class, level                 int
	attrs                              [7]int
	skills                             [41]int
	main, off                          FightWeapon
	hasMain, hasOff                    bool
	haste                              bool
	mode, offMode, monster, difficulty int
}

func rankPlan(w *FightWorld, t RankTemplate) ([]RankRow, map[string]int, []rankJob, int, error) {
	if w == nil || w.x == nil {
		return nil, nil, nil, 0, fmt.Errorf("rank: missing world")
	}
	if t.AttrMin < 0 || t.AttrMin > 125 {
		return nil, nil, nil, 0, fmt.Errorf("rank: AttrMin must be 0..125")
	}
	if _, err := rankIDs(t.Attrs, 7, "attribute"); err != nil {
		return nil, nil, nil, 0, err
	}
	if len(t.Levels) == 0 {
		return nil, nil, nil, 0, fmt.Errorf("rank: Levels must not be empty")
	}
	for _, lv := range t.Levels {
		if lv < 1 {
			return nil, nil, nil, 0, fmt.Errorf("rank: level must be positive")
		}
	}
	commands, err := rankCommands(t)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	skillPrios, err := rankSkillPrios(t)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	for s, v := range t.Train {
		if s < 0 || s > 40 || v < 0 || v > 100 {
			return nil, nil, nil, 0, fmt.Errorf("rank: Train id 0..40, value 0..100")
		}
	}
	loadouts := t.Loadouts
	if loadouts != nil {
		if t.Main != nil || t.Off != nil {
			return nil, nil, nil, 0, fmt.Errorf("rank: Loadouts cannot be combined with Main/Off")
		}
	} else {
		loadouts = []RankLoadout{{Name: "-", Main: t.Main, Off: t.Off}}
	}
	for _, l := range loadouts {
		if l.Main == nil && l.Off != nil {
			return nil, nil, nil, 0, fmt.Errorf("rank: unarmed Main cannot have Off")
		}
		if strings.ContainsAny(l.Name, "\t\r\n") {
			return nil, nil, nil, 0, fmt.Errorf("rank: loadout name contains TSV delimiter")
		}
		for _, wp := range []*FightWeapon{l.Main, l.Off} {
			if wp != nil && (wp.Item < 0 || wp.Item >= w.nItem || wp.Skill < 0 || wp.Skill > 40) {
				return nil, nil, nil, 0, fmt.Errorf("rank: invalid weapon")
			}
		}
		for _, sp := range skillPrios {
			if _, err := rankExpandSkills(w, l, sp); err != nil {
				return nil, nil, nil, 0, err
			}
		}
	}
	if t.Mode < 0 || t.Mode > 8 || t.OffMode < 0 || t.OffMode > 8 || t.Difficulty < 0 || t.Difficulty > 3 || t.Monster < 0 || t.Monster >= w.nMon {
		return nil, nil, nil, 0, fmt.Errorf("rank: invalid mode, difficulty or monster")
	}
	if err := rankMetricOptions(t); err != nil {
		return nil, nil, nil, 0, err
	}
	monsters := []int{t.Monster}
	if len(t.Monsters) > 0 {
		if _, err := rankIDs(t.Monsters, w.nMon, "monster"); err != nil {
			return nil, nil, nil, 0, err
		}
		monsters = append([]int(nil), t.Monsters...)
	}
	monsterKey := fmt.Sprint(monsters)
	races, err := rankIDs(t.Races, 11, "race")
	if err != nil {
		return nil, nil, nil, 0, err
	}
	classes, err := rankIDs(t.Classes, 15, "class")
	if err != nil {
		return nil, nil, nil, 0, err
	}
	var perms [][]int
	p := append([]int(nil), t.Attrs...)
	var visit func(int)
	visit = func(i int) {
		if i == len(p) {
			perms = append(perms, append([]int(nil), p...))
			return
		}
		for j := i; j < len(p); j++ {
			p[i], p[j] = p[j], p[i]
			visit(i + 1)
			p[i], p[j] = p[j], p[i]
		}
	}
	visit(0)
	var rows []RankRow
	var jobs []rankJob
	cache := make(map[rankKey]int)
	cells := 0
	skipped := make(map[string]int)
	for _, race := range races {
		for _, class := range classes {
			for _, l := range loadouts {
				if _, err := PlanAttrs(w.x, race, class, 1, t.Attrs); err != nil {
					skipped["creation"]++
					continue
				}
				reason, note := rankEquipment(w, race, class, l)
				if reason != "" {
					skipped[reason]++
					if l.Main != nil {
						continue
					}
				}
				for _, command := range commands {
					if command == "berserk" && !rankBerserk(w, class, l.Main) {
						skipped["berserk"]++
						continue
					}
					for _, sp := range skillPrios {
						sprio, err := rankExpandSkills(w, l, sp)
						if err != nil {
							return nil, skipped, nil, 0, err
						}
						for _, prio := range perms {
							row := RankRow{Command: command, Race: race, Class: class, Loadout: l.Name, Note: note, Prio: append([]int(nil), prio...), SkillPrio: sprio}
							for _, lv := range t.Levels {
								cells++
								raw, err := PlanSkills(w.x, race, class, lv, prio, sprio)
								if err != nil {
									return nil, skipped, nil, 0, err
								}
								skills := make(map[int]int, len(raw))
								for s, v := range raw {
									if w.x.Avail[s][class] == 0 {
										continue // Train cannot give a skill the class lacks (0x615840)
									}
									skills[s] = min(100, v+t.Train[s])
								}
								attrs, err := PlanAttrs(w.x, race, class, lv, prio)
								if err != nil {
									return nil, skipped, nil, 0, err
								}
								for i := range attrs {
									attrs[i] = max(attrs[i], t.AttrMin)
								}
								attrs = fightHasteAttrs(attrs, t.Haste)
								if reason != "" || !rankRequirements(w, l, attrs, skills) {
									row.Damage = append(row.Damage, math.Inf(-1))
									continue
								}
								key := rankKey{monsters: monsterKey, metric: t.Metric, command: command, race: race, class: class, level: lv, attrs: attrs, haste: t.Haste, mode: t.Mode, offMode: t.OffMode, monster: t.Monster, difficulty: t.Difficulty}
								if l.Main != nil {
									key.hasMain = true
									key.main = *l.Main
									key.main.Skill = int(w.items[l.Main.Item*0x10d+0x46])
								}
								for id, v := range skills {
									key.skills[id] = v
								}
								if l.Off != nil {
									key.hasOff = true
									key.off = *l.Off
									key.off.Skill = int(w.items[l.Off.Item*0x10d+0x46])
								}
								idx, ok := cache[key]
								if !ok {
									idx = len(jobs)
									cache[key] = idx
									jobs = append(jobs, rankJob{monsters: monsters, metric: t.Metric, command: command, scenario: FightScenario{Race: race, Class: class, Level: lv, Prio: prio, Attrs: &attrs, Haste: t.Haste, Skills: skills, Main: l.Main, Off: l.Off, Mode: t.Mode, OffMode: t.OffMode, Monster: t.Monster, Difficulty: t.Difficulty}})
								}
								jobs[idx].dest = append(jobs[idx].dest, [2]int{len(rows), len(row.Damage)})
								row.Damage = append(row.Damage, 0)
							}
							rows = append(rows, row)
						}
					}
				}
			}
		}
	}
	return rows, skipped, jobs, cells, nil
}

// RankPareto retains equal points; dominance requires at least one strict gain.
func RankPareto(rows []RankRow) []RankRow {
	var front []RankRow
	for i, a := range rows {
		dominated := false
		for j, b := range rows {
			if i == j {
				continue
			}
			all, strict := true, false
			for k, v := range a.Damage {
				if b.Damage[k] < v {
					all = false
					break
				}
				strict = strict || b.Damage[k] > v
			}
			if all && strict {
				dominated = true
				break
			}
		}
		if !dominated {
			front = append(front, a)
		}
	}
	return front
}

// WriteRank emits TSV, a complete Pareto section, and the skipped combination count.
func WriteRank(out io.Writer, w *FightWorld, t RankTemplate, top int) error {
	if top < 0 {
		return fmt.Errorf("rank: top must be nonnegative")
	}
	return WriteRankWorkers(out, w, t, top, runtime.GOMAXPROCS(0))
}

// WriteRankWorkers evaluates unique inputs with a fixed-size worker pool.
func WriteRankWorkers(out io.Writer, w *FightWorld, t RankTemplate, top, workers int) error {
	if top < 0 {
		return fmt.Errorf("rank: top must be nonnegative")
	}
	rows, skipped, fights, cells, err := rankWorkers(w, t, workers)
	if err != nil {
		return err
	}
	return writeRankRows(out, t.Levels, rows, skipped, top, fights, cells)
}

func writeRankRows(out io.Writer, levels []int, rows []RankRow, skipped map[string]int, top int, counts ...int) error {
	rows = append([]RankRow(nil), rows...)
	front := RankPareto(rows)
	if top > 0 {
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Damage[len(levels)-1] > rows[j].Damage[len(levels)-1] })
		rows = rows[:min(top, len(rows))]
	}
	var b strings.Builder
	b.WriteString("race\tclass\tloadout\tcommand\tprio\tsprio")
	for _, lv := range levels {
		fmt.Fprintf(&b, "\tdmg_L%d", lv)
	}
	b.WriteString("\tnote\n")
	write := func(rs []RankRow) {
		for _, r := range rs {
			ps := make([]string, len(r.Prio))
			for i, v := range r.Prio {
				ps[i] = strconv.Itoa(v)
			}
			fmt.Fprintf(&b, "%d\t%d\t%s\t%s\t%s", r.Race, r.Class, r.Loadout, r.Command, strings.Join(ps, ","))
			ss := make([]string, len(r.SkillPrio))
			for i, v := range r.SkillPrio {
				ss[i] = strconv.Itoa(v)
			}
			fmt.Fprintf(&b, "\t%s", strings.Join(ss, ","))
			for _, d := range r.Damage {
				if math.IsInf(d, -1) {
					b.WriteString("\tNA")
				} else {
					fmt.Fprintf(&b, "\t%.17g", d)
				}
			}
			fmt.Fprintf(&b, "\t%s\n", r.Note)
		}
	}
	write(rows)
	b.WriteString("\n# pareto\n")
	write(front)
	if len(counts) == 2 {
		fmt.Fprintf(&b, "# fights %d %d\n", counts[0], counts[1])
	}
	reasons := make([]string, 0, len(skipped))
	for reason := range skipped {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	for _, reason := range reasons {
		fmt.Fprintf(&b, "# skipped %s %d\n", reason, skipped[reason])
	}
	_, err := io.WriteString(out, b.String())
	return err
}

// Equipment failures count once per race/class/loadout, using the first reason.
func rankEquipment(w *FightWorld, race, class int, l RankLoadout) (string, string) {
	if l.Main == nil {
		if w.x.Avail[14][class] == 0 {
			return "weapon-skill", "技能 14 不可用"
		}
		return "", ""
	}
	gender := byte(3)
	for _, wp := range []*FightWeapon{l.Main, l.Off} {
		if wp == nil {
			continue
		}
		r := w.items[wp.Item*0x10d : (wp.Item+1)*0x10d]
		if binary.LittleEndian.Uint16(r[0x76:])&(1<<class) == 0 {
			return "class-mask", ""
		}
		if binary.LittleEndian.Uint32(r[0x78:])&(1<<race) == 0 {
			return "race-mask", ""
		}
		skill := int(r[0x46])
		if skill >= 41 || w.x.Avail[skill][class] == 0 {
			return "weapon-skill", ""
		}
		gender &= r[0x7c]
	}
	if l.Off != nil {
		if w.items[l.Main.Item*0x10d+0x41]&4 != 0 {
			return "two-handed", ""
		}
		if w.items[l.Off.Item*0x10d+0x41]&8 == 0 {
			return "offhand", ""
		}
	}
	note := ""
	if gender == 1 || gender == 2 {
		note = "此武器組限單一性別"
	}
	return "", note
}

func rankRequirements(w *FightWorld, l RankLoadout, attrs [7]int, skills map[int]int) bool {
	for _, wp := range []*FightWeapon{l.Main, l.Off} {
		if wp == nil {
			continue
		}
		r := w.items[wp.Item*0x10d : (wp.Item+1)*0x10d]
		for _, off := range []int{0x7d, 0x7f, 0x81, 0x83} {
			id, need := int(r[off]), int(r[off+1])
			if id == 255 || need == 0 {
				continue
			}
			if off < 0x81 {
				if id >= len(attrs) || attrs[id] < need {
					return false
				}
			} else if id >= 41 || skills[id] < need {
				return false
			}
		}
	}
	return true
}

// At most one non-null priority source; none means an empty SkillPrio
// (Go callers). JSON templates must name exactly one (ReadRankTemplate).
func rankSkillPrios(t RankTemplate) ([][]int, error) {
	n := 0
	for _, present := range []bool{t.SkillPrio != nil, t.SkillSet != nil, t.SkillPrios != nil} {
		if present {
			n++
		}
	}
	if n > 1 {
		return nil, fmt.Errorf("rank: give exactly one of SkillPrio, SkillSet, SkillPrios")
	}
	ps := t.SkillPrios
	if n == 0 {
		ps = [][]int{nil}
	}
	if t.SkillPrio != nil {
		ps = [][]int{t.SkillPrio}
	}
	if t.SkillSet != nil {
		ps = [][]int{t.SkillSet}
	}
	for _, p := range ps {
		seen := map[int]bool{}
		for _, id := range p {
			if id < -2 || id > 40 || seen[id] {
				return nil, fmt.Errorf("rank: invalid or duplicate skill id %d", id)
			}
			seen[id] = true
		}
	}
	if t.SkillSet == nil {
		return ps, nil
	}
	var result [][]int
	p := append([]int(nil), t.SkillSet...)
	var visit func(int)
	visit = func(i int) {
		if i == len(p) {
			result = append(result, append([]int{}, p...))
			return
		}
		for j := i; j < len(p); j++ {
			p[i], p[j] = p[j], p[i]
			visit(i + 1)
			p[i], p[j] = p[j], p[i]
		}
	}
	visit(0)
	return result, nil
}

func rankExpandSkills(w *FightWorld, l RankLoadout, p []int) ([]int, error) {
	var result []int
	seen := map[int]bool{}
	for _, id := range p {
		if id == -1 {
			id = 14
			if l.Main != nil {
				id = int(w.items[l.Main.Item*0x10d+0x46])
			}
		}
		if id == -2 {
			if l.Off == nil {
				return nil, fmt.Errorf("rank: {off} requires an offhand weapon")
			}
			id = int(w.items[l.Off.Item*0x10d+0x46])
		}
		if id > 40 {
			return nil, fmt.Errorf("rank: weapon skill outside 0..40")
		}
		if !seen[id] {
			result = append(result, id)
			seen[id] = true
		}
	}
	return result, nil
}

// RankModeDistribution implements B15 over sorted enabled bits. A nil mask
// means an unarmed hand; unarmedSkill is the effective skill, including bonus.
func RankModeDistribution(mask *uint16, unarmedSkill int) ([9]float64, error) {
	var result [9]float64
	bits := uint16(1 << 5)
	if mask != nil {
		bits = *mask & 0x1ff
	} else if unarmedSkill >= 5 {
		bits |= 1 << 6
	}
	var modes []int
	for m := 0; m < 9; m++ {
		if bits&(1<<m) != 0 {
			modes = append(modes, m)
		}
	}
	if len(modes) == 0 {
		return result, fmt.Errorf("rank: weapon has no attack modes")
	}
	for j, m := range modes {
		result[m] = float64((8-j)/len(modes)+1) / 9
	}
	return result, nil
}

func rankCommands(t RankTemplate) ([]string, error) {
	if t.Commands == nil {
		return []string{""}, nil // Legacy fixed-mode templates and Go callers.
	}
	if t.Mode != 0 || t.OffMode != 0 {
		return nil, fmt.Errorf("rank: Commands cannot be combined with Mode/OffMode")
	}
	seen := map[string]bool{}
	for _, command := range t.Commands {
		if (command != "attack" && command != "berserk") || seen[command] {
			return nil, fmt.Errorf("rank: invalid or duplicate command %q", command)
		}
		seen[command] = true
	}
	return t.Commands, nil
}

func rankBerserk(w *FightWorld, class int, main *FightWeapon) bool {
	for _, feature := range w.x.ClassFeat[class] {
		if feature == 0x14 {
			return main != nil && w.items[main.Item*0x10d+0x47] <= 1
		}
	}
	return false
}

func rankDamage(w *FightWorld, job *rankJob) (float64, error) {
	s := job.scenario
	var main, off [9]float64
	main[s.Mode], off[s.OffMode] = 1, 1
	if job.command != "" {
		var mask *uint16
		if s.Main != nil {
			v := binary.LittleEndian.Uint16(w.items[s.Main.Item*0x10d+0x4e:])
			mask = &v
		}
		var err error
		if job.command == "berserk" {
			main = [9]float64{3: 1}
		} else if main, err = RankModeDistribution(mask, EffectiveSkill(w.x, s.Class, 14, s.Skills[14])); err != nil {
			return 0, err
		}
		off = [9]float64{0: 1}
		if s.Main == nil {
			if off, err = RankModeDistribution(nil, EffectiveSkill(w.x, s.Class, 14, s.Skills[14])); err != nil {
				return 0, err
			}
		} else if s.Off != nil {
			v := binary.LittleEndian.Uint16(w.items[s.Off.Item*0x10d+0x4e:])
			mask = &v
			if off, err = RankModeDistribution(mask, 0); err != nil {
				return 0, err
			}
		}
	}
	monsters := job.monsters
	if len(monsters) == 0 {
		monsters = []int{s.Monster}
	}
	var score float64
	for _, monster := range monsters {
		s.Monster = monster
		hp := readDice(w.mons[monster*0x297+0xd6:])
		h := float64(hp.Z) + float64(hp.N)*(float64(hp.S)+1)/2
		var damage, instakill, swingKill float64
		var status [4]float64 // round.paralyze, round.poison, round.ko, round.poison_pool
		for i, pi := range main {
			if pi == 0 {
				continue
			}
			for j, pj := range off {
				if pj == 0 {
					continue
				}
				s.Mode, s.OffMode = i, j
				job.fights++
				kv, err := w.Fight(s, nil)
				if err != nil {
					return 0, err
				}
				foundDamage, foundKill := false, false
				hand := map[string]float64{}
				for _, v := range kv {
					hand[v.Key] = v.Val
					switch v.Key {
					case "round.damage":
						damage += pi * pj * v.Val
						foundDamage = true
					case "round.instakill":
						instakill += pi * pj * v.Val
						foundKill = true
					case "round.paralyze":
						status[0] += pi * pj * v.Val
					case "round.poison":
						status[1] += pi * pj * v.Val
					case "round.ko":
						status[2] += pi * pj * v.Val
					case "round.poison_pool":
						status[3] += pi * pj * v.Val
					}
				}
				if !foundDamage || (job.metric != "damage" && job.metric != "" && !foundKill) {
					return 0, fmt.Errorf("rank: missing round.damage or round.instakill")
				}
				swingKill += pi * pj * (hand["main.kills"] + hand["off.kills"])
			}
		}
		switch job.metric {
		case "kill-swing":
			score += swingKill
		case "kill":
			score += rankKill(h, damage, instakill)
		case "kill-hi":
			score += rankKillHi(h, damage, instakill, status)
		default:
			score += damage
		}
	}
	score /= float64(len(monsters))
	// Stabilize the final weighted score within the independent oracle tolerance.
	return strconv.ParseFloat(strconv.FormatFloat(score, 'g', 13, 64), 64)
}

// rankKill estimates kills per round from mean HP, damage and round instakill.
func rankKill(h, d, p float64) float64 {
	if d <= 0 {
		if p > 0 {
			return p
		}
		return 0
	}
	n := 1.0
	if h > 0 {
		n = math.Ceil(h / d)
	}
	return rankKillN(n, p)
}

// rankKillHi is the effects upper bound (weapon-effects.md N1 [未取得] items at
// their top): once paralyzed or KO'd the target stays so and takes double damage
// (the doubling itself is verified: ko-paralyze-doubling.md; still ~x2, dice groups only),
// once poisoned it loses the full pool P every round. Round j then deals
// d*(2-(1-s)^j) + P*(1-(1-q)^j) with s = 1-(1-paralyze)(1-ko), q = poison.
// The lower bound is rankKill (no effect gain).
func rankKillHi(h, d, p float64, st [4]float64) float64 {
	s := 1 - (1-st[0])*(1-st[2])
	q, pool := st[1], st[3]
	if d <= 0 && (q == 0 || pool == 0) || h <= 0 {
		return rankKill(h, d, p)
	}
	n, sum := 0.0, 0.0
	for sum < h {
		n++
		sum += d*(2-math.Pow(1-s, n)) + pool*(1-math.Pow(1-q, n))
	}
	return rankKillN(n, p)
}

// rankKillN is kills per round when n rounds of damage kill and each round instakills with p.
func rankKillN(n, p float64) float64 {
	if p == 0 {
		return 1 / n
	}
	if p == 1 {
		return 1
	}
	// Closed form of the finite survival sum; stable even for small p or large n.
	e := -math.Expm1(n*math.Log1p(-p)) / p
	return 1 / e
}
