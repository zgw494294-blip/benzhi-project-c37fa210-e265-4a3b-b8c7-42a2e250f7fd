package domain

import (
	"fmt"
	"sort"
	"strings"
)

func NormalizeCues(cues []CaptionCue) ([]CaptionCue, error) {
	if len(cues) == 0 {
		return nil, fmt.Errorf("%w: 字幕不能为空", ErrInvalidInput)
	}
	out := make([]CaptionCue, len(cues))
	copy(out, cues)
	for i := range out {
		out[i].SourceText = strings.TrimSpace(out[i].SourceText)
		out[i].TranslatedText = strings.TrimSpace(out[i].TranslatedText)
		if out[i].Sequence <= 0 {
			return nil, fmt.Errorf("%w: 序号必须为正数", ErrInvalidInput)
		}
		if out[i].OutMillis <= out[i].InMillis {
			return nil, fmt.Errorf("%w: 出点必须晚于入点", ErrInvalidInput)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	for i, c := range out {
		if c.Sequence != i+1 {
			return nil, fmt.Errorf("%w: 序号不连续", ErrInvalidInput)
		}
	}
	return out, nil
}

func (c CaptionCue) Duration() int64 { return c.OutMillis - c.InMillis }
func (c CaptionCue) IsEmpty() bool   { return strings.TrimSpace(c.TranslatedText) == "" }
func (c CaptionCue) Contains(text string, caseSensitive bool) bool {
	if !caseSensitive {
		return strings.Contains(strings.ToLower(c.SourceText), strings.ToLower(text))
	}
	return strings.Contains(c.SourceText, text)
}
func (c CaptionCue) TranslationContains(text string, caseSensitive bool) bool {
	if !caseSensitive {
		return strings.Contains(strings.ToLower(c.TranslatedText), strings.ToLower(text))
	}
	return strings.Contains(c.TranslatedText, text)
}
func CueDigest(c CaptionCue) string {
	return Digest(struct {
		S                            int
		I, O                         int64
		Source, Translation, Speaker string
	}{c.Sequence, c.InMillis, c.OutMillis, c.SourceText, c.TranslatedText, c.Speaker})
}

func DiffCues(before, after []CaptionCue) []CueChange {
	left := make(map[int]CaptionCue, len(before))
	right := make(map[int]CaptionCue, len(after))
	sequences := map[int]bool{}
	for _, cue := range before {
		left[cue.Sequence] = cue
		sequences[cue.Sequence] = true
	}
	for _, cue := range after {
		right[cue.Sequence] = cue
		sequences[cue.Sequence] = true
	}
	keys := make([]int, 0, len(sequences))
	for sequence := range sequences {
		keys = append(keys, sequence)
	}
	sort.Ints(keys)
	out := make([]CueChange, 0)
	for _, sequence := range keys {
		oldCue, oldOK := left[sequence]
		newCue, newOK := right[sequence]
		if !oldOK {
			cue := newCue
			out = append(out, CueChange{Sequence: sequence, Kind: CueAdded, After: &cue})
			continue
		}
		if !newOK {
			cue := oldCue
			out = append(out, CueChange{Sequence: sequence, Kind: CueDeleted, Before: &cue})
			continue
		}
		fields := diffCueFields(oldCue, newCue)
		if len(fields) > 0 {
			oldCopy, newCopy := oldCue, newCue
			out = append(out, CueChange{Sequence: sequence, Kind: CueModified, Before: &oldCopy, After: &newCopy, Fields: fields})
		}
	}
	return out
}

func diffCueFields(before, after CaptionCue) []CueFieldChange {
	out := make([]CueFieldChange, 0, 5)
	if before.Sequence != after.Sequence {
		out = append(out, CueFieldChange{Field: "sequence", Before: before.Sequence, After: after.Sequence})
	}
	if before.InMillis != after.InMillis {
		out = append(out, CueFieldChange{Field: "inMillis", Before: before.InMillis, After: after.InMillis})
	}
	if before.OutMillis != after.OutMillis {
		out = append(out, CueFieldChange{Field: "outMillis", Before: before.OutMillis, After: after.OutMillis})
	}
	if before.SourceText != after.SourceText {
		out = append(out, CueFieldChange{Field: "sourceText", Before: before.SourceText, After: after.SourceText})
	}
	if before.TranslatedText != after.TranslatedText {
		out = append(out, CueFieldChange{Field: "translatedText", Before: before.TranslatedText, After: after.TranslatedText})
	}
	if before.Speaker != after.Speaker {
		out = append(out, CueFieldChange{Field: "speaker", Before: before.Speaker, After: after.Speaker})
	}
	return out
}
