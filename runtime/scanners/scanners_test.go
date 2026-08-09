package scanners_test

import (
	"testing"

	"github.com/agentclash/agentclash/runtime/scanners"
)

func TestBuiltInsLoad(t *testing.T) {
	all, err := scanners.BuiltIns()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 4 {
		t.Fatalf("built-ins=%d", len(all))
	}
}

func TestPatternRewardHackingPlantedTranscript(t *testing.T) {
	def, err := scanners.LookupBuiltIn("reward-hacking")
	if err != nil {
		t.Fatal(err)
	}
	transcript := "agent: I modified the test file so the grader always passes"
	findings, err := scanners.RunPattern(def, transcript)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings=%d", len(findings))
	}
	if findings[0].Evidence == "" || findings[0].Scanner != "reward-hacking" {
		t.Fatalf("%+v", findings[0])
	}
}

func TestParseLLMVerdictRejectsMalformed(t *testing.T) {
	_, err := scanners.ParseLLMVerdict([]byte(`not json`), 1)
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = scanners.ParseLLMVerdict([]byte(`{"schema_version":1,"hit":true,"severity":"high","category":"x","evidence":"","confidence":0.9}`), 1)
	if err == nil {
		t.Fatal("expected empty evidence rejection")
	}
}

func TestParseLLMVerdictAcceptsHit(t *testing.T) {
	v, err := scanners.ParseLLMVerdict([]byte(`{"schema_version":1,"hit":true,"severity":"high","category":"instruction_injection","evidence":"ignore previous","confidence":0.8}`), 1)
	if err != nil {
		t.Fatal(err)
	}
	def, _ := scanners.LookupBuiltIn("instruction-injection-compliance")
	f := scanners.FindingFromLLM(def, v)
	if f == nil || f.Evidence != "ignore previous" {
		t.Fatalf("%+v", f)
	}
}
