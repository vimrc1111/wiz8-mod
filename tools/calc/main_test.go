package main

import (
	"errors"
	"strings"
	"testing"

	"wiz8-mod/internal/calc"
)

// grok 稽核 cXan F3：技能值超出 0..125 要拒收，同 melee。
func TestBuildScenarioSkillRange(t *testing.T) {
	for _, v := range []string{"-1", "126"} {
		_, err := readBuildScenario(strings.NewReader(`{"Main":{"Item":0,"Skill":0},"Skills":{"0":` + v + `}}`))
		if err == nil {
			t.Errorf("Skills value %s accepted", v)
		}
	}
	if _, err := readBuildScenario(strings.NewReader(`{"Main":{"Item":0,"Skill":0},"Skills":{"0":125}}`)); err != nil {
		t.Errorf("Skills value 125 rejected: %v", err)
	}
}

func TestNodeCLI(t *testing.T) {
	var out strings.Builder
	if err := run([]string{"f", "00517970", "2", "3", "6"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"dist\n", "5: ", "20: ", "mean value:", "in: Z, N, S", "out: value"} {
		if !strings.Contains(out.String(), part) {
			t.Errorf("missing %q: %s", part, out.String())
		}
	}
	for _, args := range [][]string{{"f"}, {"f", "unknown"}, {"f", "00517970", "2"}, {"f", "00517970", "x", "1", "6"}, {"f", "00517970", "2", "256", "6"}} {
		if err := run(args, &out); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}

func TestPartyCLIInvalid(t *testing.T) {
	for _, args := range [][]string{{"party"}, {"party", "-metric", "attr:0"}, {"party", "-exe", "missing", "-metric", "bad:0"}, {"party", "-metric"}, {"party", "-unknown"}, {"party", "-exe", "missing", "-metric", "attr:0", "extra"}, {"party", "-exe", "missing", "-metric", "attr:0"}} {
		var out strings.Builder
		if err := run(args, &out); err == nil {
			t.Errorf("accepted %v", args)
		}
		if out.Len() != 0 {
			t.Errorf("error polluted stdout: %q", out.String())
		}
	}
}

type partyBrokenWriter struct{}

func (partyBrokenWriter) Write(p []byte) (int, error) { return 0, errors.New("write failed") }

func TestPartyTSV(t *testing.T) {
	x := &calc.ExeTables{}
	ms, err := calc.ParsePartyMetrics("attr:0,l100:6,attr:0")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := writeParty(&out, x, ms); err != nil {
		t.Fatal(err)
	}
	want := "race\tclass\tgender\tattrs\tattr:0\tl100:6\tattr:0\n0\t0\t0\t20,0,0,0,0,0,20\t20\t28\t20\n"
	if out.String() != want {
		t.Fatalf("TSV=%q want=%q", out.String(), want)
	}
	if err := writeParty(partyBrokenWriter{}, x, ms); err == nil {
		t.Error("write error swallowed")
	}
}
