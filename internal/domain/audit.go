package domain

import (
	"sort"
	"time"
)

type AuditRecord struct {
	Sequence int64     `json:"sequence"`
	Type     string    `json:"type"`
	Actor    string    `json:"actor"`
	At       time.Time `json:"at"`
	Digest   string    `json:"digest"`
}

func SortFindings(findings []ValidationFinding) []ValidationFinding {
	out := append(make([]ValidationFinding, 0, len(findings)), findings...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RuleCode != out[j].RuleCode {
			return out[i].RuleCode < out[j].RuleCode
		}
		if out[i].CueSequence != out[j].CueSequence {
			return out[i].CueSequence < out[j].CueSequence
		}
		return out[i].ID < out[j].ID
	})
	return out
}
func OpenBlocking(findings []ValidationFinding) []ValidationFinding {
	out := []ValidationFinding{}
	for _, f := range findings {
		if f.Severity == SeverityBlocking && f.Status == FindingOpen {
			out = append(out, f)
		}
	}
	return SortFindings(out)
}
func HasOpenFindings(findings []ValidationFinding) bool {
	for _, f := range findings {
		if f.Status == FindingOpen {
			return true
		}
	}
	return false
}
