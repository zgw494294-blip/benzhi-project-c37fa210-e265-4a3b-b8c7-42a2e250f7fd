package application

import (
	"fmt"
	"stagecaption-finalizer/internal/domain"
)

func ValidationText(run domain.ValidationRun) string {
	summary := SummarizeFindings(run.Findings)
	return fmt.Sprintf("规则摘要 %s；共 %d 项问题，其中阻断 %d 项，待处理 %d 项，已关闭 %d 项", run.RuleDigest, summary.Total, summary.Blocking, summary.Open, summary.Resolved)
}
func CanExport(m *domain.FreezeManifest) bool { return m != nil && m.Verify() }
func ReviewDecisionText(d domain.ReviewDecision) string {
	if d.Decision == "approve" {
		return "校审员已批准当前修订"
	}
	return "校审员已退回当前修订：" + d.Reason
}
