package validation

import (
	"fmt"
	"strings"

	"stagecaption-finalizer/internal/domain"
)

type findingAdder func(string, domain.FindingSeverity, int, string, string)

func validateTiming(p domain.CaptionProject, r domain.CaptionRevision, add findingAdder) {
	var previous *domain.CaptionCue
	for i := range r.Cues {
		cue := r.Cues[i]
		expected := i + 1
		if cue.Sequence != expected {
			add("CUE_SEQUENCE", domain.SeverityBlocking, cue.Sequence, "cue 序号必须从 1 连续递增", fmt.Sprintf("expected=%d actual=%d", expected, cue.Sequence))
		}
		if cue.InMillis < 0 || cue.OutMillis <= cue.InMillis {
			add("TIMECODE_RANGE", domain.SeverityBlocking, cue.Sequence, "入点必须非负且出点晚于入点", fmt.Sprintf("in=%d out=%d", cue.InMillis, cue.OutMillis))
		}
		duration := cue.OutMillis - cue.InMillis
		if duration < p.MinDisplayMillis || duration > p.MaxDisplayMillis {
			add("DISPLAY_DURATION", domain.SeverityBlocking, cue.Sequence, "显示时长超出项目规则", fmt.Sprintf("duration=%d allowed=%d..%d", duration, p.MinDisplayMillis, p.MaxDisplayMillis))
		}
		if strings.TrimSpace(cue.TranslatedText) == "" {
			add("EMPTY_TRANSLATION", domain.SeverityBlocking, cue.Sequence, "译文不能为空", "translatedText is blank")
		}
		if previous != nil && cue.InMillis < previous.OutMillis {
			add("CUE_OVERLAP", domain.SeverityBlocking, cue.Sequence, "相邻字幕时码重叠", fmt.Sprintf("previousOut=%d currentIn=%d", previous.OutMillis, cue.InMillis))
		}
		previous = &r.Cues[i]
	}
	if len(r.Cues) == 0 {
		add("EMPTY_REVISION", domain.SeverityBlocking, 0, "修订至少包含一条字幕", "cueCount=0")
	}
}
