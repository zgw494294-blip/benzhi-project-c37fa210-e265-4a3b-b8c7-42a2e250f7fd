package domain

import "time"

type ProjectStatus string

const (
	StatusDraft     ProjectStatus = "draft"
	StatusNeedsFix  ProjectStatus = "needs_fix"
	StatusValidated ProjectStatus = "validated"
	StatusInReview  ProjectStatus = "in_review"
	StatusReturned  ProjectStatus = "returned"
	StatusApproved  ProjectStatus = "approved"
	StatusFrozen    ProjectStatus = "frozen"
)

type CaptionProject struct {
	ID                string        `json:"id"`
	Title             string        `json:"title"`
	SourceLanguage    string        `json:"sourceLanguage"`
	TargetLanguage    string        `json:"targetLanguage"`
	FrameRate         float64       `json:"frameRate"`
	MinDisplayMillis  int64         `json:"minDisplayMillis"`
	MaxDisplayMillis  int64         `json:"maxDisplayMillis"`
	Status            ProjectStatus `json:"status"`
	CurrentRevisionID string        `json:"currentRevisionID,omitempty"`
	Version           int64         `json:"version"`
	GlossaryVersion   int64         `json:"glossaryVersion"`
	CreatedAt         time.Time     `json:"createdAt"`
	UpdatedAt         time.Time     `json:"updatedAt"`
}

type GlossaryTerm struct {
	ID                    string    `json:"id"`
	ProjectID             string    `json:"projectID"`
	SourceText            string    `json:"sourceText"`
	RequiredTranslation   string    `json:"requiredTranslation"`
	ForbiddenTranslations []string  `json:"forbiddenTranslations"`
	CaseSensitive         bool      `json:"caseSensitive"`
	Version               int64     `json:"version"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type CaptionCue struct {
	Sequence       int    `json:"sequence"`
	InMillis       int64  `json:"inMillis"`
	OutMillis      int64  `json:"outMillis"`
	SourceText     string `json:"sourceText"`
	TranslatedText string `json:"translatedText"`
	Speaker        string `json:"speaker,omitempty"`
}

type CaptionRevision struct {
	ID               string       `json:"id"`
	ProjectID        string       `json:"projectID"`
	ParentRevisionID string       `json:"parentRevisionID,omitempty"`
	RevisionNumber   int          `json:"revisionNumber"`
	SubmittedBy      string       `json:"submittedBy"`
	SubmittedAt      time.Time    `json:"submittedAt"`
	GlossaryVersion  int64        `json:"glossaryVersion"`
	Summary          string       `json:"summary"`
	ContentDigest    string       `json:"contentDigest"`
	Cues             []CaptionCue `json:"cues"`
}

type FindingSeverity string
type FindingStatus string

const (
	SeverityBlocking FindingSeverity = "blocking"
	SeverityWarning  FindingSeverity = "warning"
	FindingOpen      FindingStatus   = "open"
	FindingResolved  FindingStatus   = "resolved"
)

type ValidationFinding struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"projectID"`
	RevisionID     string          `json:"revisionID"`
	RuleCode       string          `json:"ruleCode"`
	Severity       FindingSeverity `json:"severity"`
	CueSequence    int             `json:"cueSequence"`
	Message        string          `json:"message"`
	Evidence       string          `json:"evidence"`
	ResolutionNote string          `json:"resolutionNote,omitempty"`
	Status         FindingStatus   `json:"status"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type ValidationRun struct {
	ID              string              `json:"id"`
	RevisionID      string              `json:"revisionID"`
	GlossaryVersion int64               `json:"glossaryVersion"`
	RuleDigest      string              `json:"ruleDigest"`
	RuleSummary     DisplayRules        `json:"ruleSummary"`
	Findings        []ValidationFinding `json:"findings"`
	RanAt           time.Time           `json:"ranAt"`
}

type CueChangeKind string

const (
	CueAdded    CueChangeKind = "added"
	CueDeleted  CueChangeKind = "deleted"
	CueModified CueChangeKind = "modified"
)

type CueFieldChange struct {
	Field  string `json:"field"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

type CueChange struct {
	Sequence int              `json:"sequence"`
	Kind     CueChangeKind    `json:"kind"`
	Before   *CaptionCue      `json:"before,omitempty"`
	After    *CaptionCue      `json:"after,omitempty"`
	Fields   []CueFieldChange `json:"fields,omitempty"`
}

type ReviewDecision struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"projectID"`
	RevisionID string    `json:"revisionID"`
	Reviewer   string    `json:"reviewer"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason,omitempty"`
	DecidedAt  time.Time `json:"decidedAt"`
}

type FreezeManifest struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectID"`
	RevisionID       string    `json:"revisionID"`
	ReviewDecisionID string    `json:"reviewDecisionID"`
	RuleDigest       string    `json:"ruleDigest"`
	GlossaryDigest   string    `json:"glossaryDigest"`
	CaptionDigest    string    `json:"captionDigest"`
	AuditHeadDigest  string    `json:"auditHeadDigest"`
	VerificationCode string    `json:"verificationCode"`
	FrozenBy         string    `json:"frozenBy"`
	FrozenAt         time.Time `json:"frozenAt"`
}
