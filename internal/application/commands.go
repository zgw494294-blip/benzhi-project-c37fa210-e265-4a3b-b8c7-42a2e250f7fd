package application

import (
	"time"

	"stagecaption-finalizer/internal/domain"
)

type CreateProjectCommand struct {
	Title            string  `json:"title"`
	SourceLanguage   string  `json:"sourceLanguage"`
	TargetLanguage   string  `json:"targetLanguage"`
	FrameRate        float64 `json:"frameRate"`
	MinDisplayMillis int64   `json:"minDisplayMillis"`
	MaxDisplayMillis int64   `json:"maxDisplayMillis"`
	Actor            string  `json:"actor"`
	IdempotencyKey   string  `json:"idempotencyKey"`
}
type TermCommand struct {
	SourceText            string   `json:"sourceText"`
	RequiredTranslation   string   `json:"requiredTranslation"`
	ForbiddenTranslations []string `json:"forbiddenTranslations"`
	CaseSensitive         bool     `json:"caseSensitive"`
	ExpectedVersion       int64    `json:"expectedVersion"`
	Actor                 string   `json:"actor"`
	IdempotencyKey        string   `json:"idempotencyKey"`
}
type RuleUpdateCommand struct {
	Title            string  `json:"title"`
	SourceLanguage   string  `json:"sourceLanguage"`
	TargetLanguage   string  `json:"targetLanguage"`
	FrameRate        float64 `json:"frameRate"`
	MinDisplayMillis int64   `json:"minDisplayMillis"`
	MaxDisplayMillis int64   `json:"maxDisplayMillis"`
	ExpectedVersion  int64   `json:"expectedVersion"`
	Actor            string  `json:"actor"`
	IdempotencyKey   string  `json:"idempotencyKey"`
}
type GlossaryEntryInput struct {
	SourceText            string   `json:"sourceText"`
	RequiredTranslation   string   `json:"requiredTranslation"`
	ForbiddenTranslations []string `json:"forbiddenTranslations"`
	CaseSensitive         bool     `json:"caseSensitive"`
}
type BatchGlossaryCommand struct {
	Entries         []GlossaryEntryInput `json:"entries"`
	ExpectedVersion int64                `json:"expectedVersion"`
	Actor           string               `json:"actor"`
	IdempotencyKey  string               `json:"idempotencyKey"`
}
type BatchGlossaryResult struct {
	GlossaryVersion int64                 `json:"glossaryVersion"`
	ImportedCount   int                   `json:"importedCount"`
	TotalCount      int                   `json:"totalCount"`
	Entries         []domain.GlossaryTerm `json:"entries"`
}
type RevisionCommand struct {
	ParentRevisionID string              `json:"parentRevisionID"`
	SubmittedBy      string              `json:"submittedBy"`
	ExpectedVersion  int64               `json:"expectedVersion"`
	IdempotencyKey   string              `json:"idempotencyKey"`
	Summary          string              `json:"summary"`
	Actor            string              `json:"actor,omitempty"`
	Cues             []domain.CaptionCue `json:"cues"`
}
type ResolveItem struct {
	FindingID      string `json:"findingID"`
	ResolutionNote string `json:"resolutionNote"`
}
type BatchResolveCommand struct {
	Items           []ResolveItem `json:"items"`
	ExpectedVersion int64         `json:"expectedVersion"`
	Actor           string        `json:"actor"`
	IdempotencyKey  string        `json:"idempotencyKey"`
}
type BatchResolveResult struct {
	RevisionID string                     `json:"revisionID"`
	Resolved   []domain.ValidationFinding `json:"resolved"`
	Version    int64                      `json:"version"`
}
type VersionedCommand struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	Actor           string `json:"actor"`
	IdempotencyKey  string `json:"idempotencyKey"`
}
type ResolveCommand struct {
	FindingID       string `json:"findingID"`
	ResolutionNote  string `json:"resolutionNote"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Actor           string `json:"actor"`
	IdempotencyKey  string `json:"idempotencyKey"`
}
type ReviewCommand struct {
	Decision        string `json:"decision"`
	Reason          string `json:"reason"`
	Reviewer        string `json:"reviewer"`
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
	Actor           string `json:"actor,omitempty"`
}

type Workspace struct {
	Project    domain.CaptionProject    `json:"project"`
	Terms      []domain.GlossaryTerm    `json:"terms"`
	Revisions  []domain.CaptionRevision `json:"revisions"`
	Validation *domain.ValidationRun    `json:"validation,omitempty"`
	Reviews    []domain.ReviewDecision  `json:"reviews"`
	Manifest   *domain.FreezeManifest   `json:"manifest,omitempty"`
}
type RevisionDiff struct {
	ProjectID string                 `json:"projectID"`
	From      domain.CaptionRevision `json:"from"`
	To        domain.CaptionRevision `json:"to"`
	Changes   []domain.CueChange     `json:"changes"`
}
type ValidationRunSummary struct {
	ID              string              `json:"id"`
	RevisionID      string              `json:"revisionID"`
	RanAt           time.Time           `json:"ranAt"`
	RuleDigest      string              `json:"ruleDigest"`
	RuleSummary     domain.DisplayRules `json:"ruleSummary"`
	GlossaryVersion int64               `json:"glossaryVersion"`
	Counts          FindingSummary      `json:"counts"`
}
type ValidationComparison struct {
	BeforeID    string                     `json:"beforeID"`
	AfterID     string                     `json:"afterID"`
	Added       []domain.ValidationFinding `json:"added"`
	Disappeared []domain.ValidationFinding `json:"disappeared"`
	Persistent  []FindingTransition        `json:"persistent"`
}
type FindingTransition struct {
	Before domain.ValidationFinding `json:"before"`
	After  domain.ValidationFinding `json:"after"`
}
type ReviewDetail struct {
	Project         domain.CaptionProject   `json:"project"`
	CurrentRevision domain.CaptionRevision  `json:"currentRevision"`
	ParentRevision  *domain.CaptionRevision `json:"parentRevision,omitempty"`
	InitialRevision bool                    `json:"initialRevision"`
	Diff            []domain.CueChange      `json:"diff"`
	Validation      *ValidationRunSummary   `json:"validation,omitempty"`
	GlossaryVersion int64                   `json:"glossaryVersion"`
	Decisions       []domain.ReviewDecision `json:"decisions"`
}
type FreezePreview struct {
	ProjectID       string                 `json:"projectID"`
	Revision        domain.CaptionRevision `json:"revision"`
	Rules           domain.DisplayRules    `json:"rules"`
	RuleDigest      string                 `json:"ruleDigest"`
	GlossaryVersion int64                  `json:"glossaryVersion"`
	Glossary        []domain.GlossaryTerm  `json:"glossary"`
	GlossaryDigest  string                 `json:"glossaryDigest"`
	CaptionDigest   string                 `json:"captionDigest"`
	Review          domain.ReviewDecision  `json:"review"`
	AuditHeadDigest string                 `json:"auditHeadDigest"`
}
type ExportBundle struct {
	ProjectID        string                `json:"projectID"`
	VerificationCode string                `json:"verificationCode"`
	Captions         string                `json:"captions"`
	Glossary         []domain.GlossaryTerm `json:"glossary"`
	Audit            any                   `json:"audit"`
	Manifest         domain.FreezeManifest `json:"manifest"`
}
type VerificationResult struct {
	Valid            bool            `json:"valid"`
	VerificationCode string          `json:"verificationCode"`
	Checks           map[string]bool `json:"checks"`
	Message          string          `json:"message"`
}
type ExportItem struct {
	ProjectID        string `json:"projectID"`
	ManifestID       string `json:"manifestID"`
	VerificationCode string `json:"verificationCode"`
	Kind             string `json:"kind"`
	ContentType      string `json:"contentType"`
	Filename         string `json:"filename"`
	Digest           string `json:"digest"`
	Content          []byte `json:"-"`
}
