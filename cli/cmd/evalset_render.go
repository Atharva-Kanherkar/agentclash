package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/agentclash/agentclash/cli/internal/output"
	"github.com/agentclash/agentclash/runtime/evalset"
)

func renderEvalsetExpansionTable(rc *RunContext, report evalset.ExpansionReport) {
	fmt.Fprintln(rc.Output.Writer(), output.Bold("Expansion"))
	rc.Output.PrintDetail("Name", report.Name)
	rc.Output.PrintDetail("Combinations", fmt.Sprintf("%d", report.Count))
	rc.Output.PrintDetail("Packs", fmt.Sprintf("%d", report.PackCount))
	rc.Output.PrintDetail("Agents", fmt.Sprintf("%d", report.AgentCount))
	rc.Output.PrintDetail("Repeats", fmt.Sprintf("%d", report.Repeats))
	if report.MaxConcurrent > 0 {
		rc.Output.PrintDetail("Max concurrent", fmt.Sprintf("%d", report.MaxConcurrent))
	}
	rows := make([][]string, 0, len(report.Combinations))
	for _, c := range report.Combinations {
		seed := ""
		if c.Seed != nil {
			seed = fmt.Sprintf("%d", *c.Seed)
		}
		rows = append(rows, []string{c.MatrixKey, c.PackRef, c.AgentRef, c.ModelRef, fmt.Sprintf("%d", c.Repeat), seed})
	}
	fmt.Fprintln(rc.Output.Writer())
	rc.Output.PrintTable([]output.Column{
		{Header: "Matrix Key"},
		{Header: "Pack"},
		{Header: "Agent"},
		{Header: "Model"},
		{Header: "Repeat"},
		{Header: "Seed"},
	}, rows)
}

func renderEvalsetList(rc *RunContext, result map[string]any) {
	items := mapSlice(result, "eval_sets")
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		set, _ := raw.(map[string]any)
		rows = append(rows, []string{
			mapString(set, "id"),
			mapString(set, "name"),
			output.StatusColor(mapString(set, "status")),
			mapString(set, "combination_count"),
			mapString(set, "created_at", "updated_at"),
		})
	}
	rc.Output.PrintTable([]output.Column{
		{Header: "ID"},
		{Header: "Name"},
		{Header: "Status"},
		{Header: "Combos"},
		{Header: "Created"},
	}, rows)
}

func renderEvalsetStatus(rc *RunContext, result map[string]any) {
	set := mapObject(result, "eval_set")
	fmt.Fprintln(rc.Output.Writer(), output.Bold("Eval Set"))
	rc.Output.PrintDetail("ID", mapString(set, "id"))
	rc.Output.PrintDetail("Name", mapString(set, "name"))
	rc.Output.PrintDetail("Status", output.StatusColor(mapString(set, "status")))
	rc.Output.PrintDetail("Combinations", mapString(set, "combination_count"))
	if sessions := mapSlice(result, "eval_session_ids"); len(sessions) > 0 {
		ids := make([]string, 0, len(sessions))
		for _, s := range sessions {
			ids = append(ids, fmt.Sprint(s))
		}
		rc.Output.PrintDetail("Sessions", strings.Join(ids, ", "))
	}
	if reason := mapString(set, "failure_reason"); reason != "" {
		rc.Output.PrintDetail("Failure", reason)
	}
	renderEvalsetReportTable(rc, result)
}

func renderEvalsetReportTable(rc *RunContext, result map[string]any) {
	agg := evalsetAggregateMap(result)
	if agg == nil {
		return
	}
	fmt.Fprintln(rc.Output.Writer())
	fmt.Fprintln(rc.Output.Writer(), output.Bold("Roll-up"))
	rc.Output.PrintDetail("Sessions", fmt.Sprintf("%v", agg["sessions"]))
	rc.Output.PrintDetail("Runs", fmt.Sprintf("%v", agg["runs"]))

	perPack := mapObject(agg, "per_pack")
	if len(perPack) > 0 {
		keys := make([]string, 0, len(perPack))
		for k := range perPack {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		rows := make([][]string, 0, len(keys))
		for _, k := range keys {
			rows = append(rows, []string{k, fmt.Sprintf("%v", perPack[k])})
		}
		fmt.Fprintln(rc.Output.Writer())
		fmt.Fprintln(rc.Output.Writer(), output.Bold("Per pack"))
		rc.Output.PrintTable([]output.Column{{Header: "Pack"}, {Header: "Sessions"}}, rows)
	}

	combos := mapSlice(agg, "combinations")
	if len(combos) == 0 {
		return
	}
	rows := make([][]string, 0, len(combos))
	for _, raw := range combos {
		m, _ := raw.(map[string]any)
		rows = append(rows, []string{
			mapString(m, "matrix_key"),
			mapString(m, "pack_ref"),
			output.StatusColor(mapString(m, "status")),
		})
	}
	fmt.Fprintln(rc.Output.Writer())
	fmt.Fprintln(rc.Output.Writer(), output.Bold("Combinations"))
	rc.Output.PrintTable([]output.Column{
		{Header: "Matrix Key"},
		{Header: "Pack"},
		{Header: "Status"},
	}, rows)

	// Simple win matrix stub: count completed vs failed by pack_ref.
	wins := map[string][2]int{}
	for _, raw := range combos {
		m, _ := raw.(map[string]any)
		pack := mapString(m, "pack_ref")
		if pack == "" || mapString(m, "matrix_key") == "" {
			continue
		}
		st := strings.ToLower(mapString(m, "status"))
		pair := wins[pack]
		if st == "completed" {
			pair[0]++
		} else if st == "failed" {
			pair[1]++
		}
		wins[pack] = pair
	}
	if len(wins) > 0 {
		keys := make([]string, 0, len(wins))
		for k := range wins {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		rows := make([][]string, 0, len(keys))
		for _, k := range keys {
			pair := wins[k]
			rows = append(rows, []string{k, fmt.Sprintf("%d", pair[0]), fmt.Sprintf("%d", pair[1])})
		}
		fmt.Fprintln(rc.Output.Writer())
		fmt.Fprintln(rc.Output.Writer(), output.Bold("Win matrix (completed vs failed runs)"))
		rc.Output.PrintTable([]output.Column{
			{Header: "Pack"},
			{Header: "Completed"},
			{Header: "Failed"},
		}, rows)
	}
}

func renderEvalsetReportCSV(rc *RunContext, result map[string]any) error {
	agg := evalsetAggregateMap(result)
	w := csv.NewWriter(rc.Output.Writer())
	_ = w.Write([]string{"matrix_key", "pack_ref", "status", "sessions", "runs"})
	sessions := fmt.Sprintf("%v", agg["sessions"])
	runs := fmt.Sprintf("%v", agg["runs"])
	for _, raw := range mapSlice(agg, "combinations") {
		m, _ := raw.(map[string]any)
		_ = w.Write([]string{
			mapString(m, "matrix_key"),
			mapString(m, "pack_ref"),
			mapString(m, "status"),
			sessions,
			runs,
		})
	}
	w.Flush()
	return w.Error()
}

func evalsetAggregateMap(result map[string]any) map[string]any {
	res := mapObject(result, "result")
	if res == nil {
		return nil
	}
	if agg := mapObject(res, "aggregate"); agg != nil {
		return agg
	}
	switch raw := res["aggregate"].(type) {
	case map[string]any:
		return raw
	case string:
		var m map[string]any
		if json.Unmarshal([]byte(raw), &m) == nil {
			return m
		}
	case []byte:
		var m map[string]any
		if json.Unmarshal(raw, &m) == nil {
			return m
		}
	case json.RawMessage:
		var m map[string]any
		if json.Unmarshal(raw, &m) == nil {
			return m
		}
	}
	return nil
}
