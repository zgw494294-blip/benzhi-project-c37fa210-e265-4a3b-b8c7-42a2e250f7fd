package domain

import "fmt"

func ParseProjectStatus(value string) (ProjectStatus, error) {
	status := ProjectStatus(value)
	switch status {
	case StatusDraft, StatusNeedsFix, StatusValidated, StatusInReview, StatusReturned, StatusApproved, StatusFrozen:
		return status, nil
	default:
		return "", fmt.Errorf("%w: 未知项目状态 %q", ErrInvalidInput, value)
	}
}
func StatusLabel(status ProjectStatus) string {
	switch status {
	case StatusDraft:
		return "草稿"
	case StatusNeedsFix:
		return "待整改"
	case StatusValidated:
		return "已校验"
	case StatusInReview:
		return "复核中"
	case StatusReturned:
		return "已退回"
	case StatusApproved:
		return "已批准"
	case StatusFrozen:
		return "已冻结"
	default:
		return "未知"
	}
}
func FindingLabel(status FindingStatus) string {
	if status == FindingResolved {
		return "已关闭"
	}
	return "待处理"
}
func SeverityLabel(level FindingSeverity) string {
	if level == SeverityBlocking {
		return "阻断"
	}
	return "提示"
}
