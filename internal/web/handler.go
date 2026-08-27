package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"stagecaption-finalizer/internal/application"
	"stagecaption-finalizer/internal/domain"
)

type Handler struct{ app *application.Service }

func New(app *application.Service) *Handler { return &Handler{app: app} }
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", ServeStatic)
	mux.HandleFunc("/workbench", ServeStatic)
	mux.HandleFunc("/static/app.css", ServeStatic)
	mux.HandleFunc("/static/app.js", ServeStatic)
	mux.HandleFunc("/api/projects", h.projects)
	mux.HandleFunc("/api/stats", h.stats)
	mux.HandleFunc("/api/projects/", h.project)
	return mux
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, h.app.Stats())
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	allowJSON(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("%w: JSON 请求体无效: %v", domain.ErrInvalidInput, err)
	}
	return nil
}
func statusError(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrVersionConflict), errors.Is(err, domain.ErrIdentityConflict), errors.Is(err, domain.ErrBlockingFindings), errors.Is(err, domain.ErrStaleValidation), errors.Is(err, domain.ErrInvalidState), errors.Is(err, domain.ErrFrozen):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}
func (h *Handler) fail(w http.ResponseWriter, err error) {
	var conflict *application.GlossaryConflictError
	if errors.As(err, &conflict) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "conflicts": conflict.Conflicts})
		return
	}
	writeJSON(w, statusError(err), map[string]string{"error": err.Error()})
}

func (h *Handler) projects(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		v, err := h.app.ListProjects()
		if err != nil {
			h.fail(w, err)
			return
		}
		writeJSON(w, http.StatusOK, v)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var c application.CreateProjectCommand
	if err := decode(r, &c); err != nil {
		h.fail(w, err)
		return
	}
	v, err := h.app.CreateProject(c)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func splitProjectPath(path string) []string {
	rest := strings.Trim(strings.TrimPrefix(path, "/api/projects/"), "/")
	if rest == "" {
		return nil
	}
	return strings.Split(rest, "/")
}

func (h *Handler) project(w http.ResponseWriter, r *http.Request) {
	parts := splitProjectPath(r.URL.Path)
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	projectID := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		v, err := h.app.GetWorkspace(projectID)
		if err != nil {
			h.fail(w, err)
			return
		}
		writeJSON(w, http.StatusOK, v)
		return
	}
	if r.Method == http.MethodGet {
		h.handleProjectGet(w, r, projectID, parts[1:])
		return
	}
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		h.handleProjectWrite(w, r, projectID, parts[1:])
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (h *Handler) handleProjectGet(w http.ResponseWriter, r *http.Request, projectID string, parts []string) {
	var out any
	var err error
	switch {
	case len(parts) == 1 && parts[0] == "export":
		out, err = h.app.Export(projectID)
		if err == nil {
			w.Header().Set("Content-Disposition", "attachment; filename=\""+application.ExportFilename(projectID, "captions.json")+"\"")
		}
	case len(parts) == 2 && parts[0] == "exports":
		h.downloadExport(w, projectID, parts[1])
		return
	case len(parts) == 1 && parts[0] == "verify":
		out, err = h.app.Verify(projectID)
	case len(parts) == 1 && parts[0] == "summary":
		out, err = h.app.Summary(projectID)
	case len(parts) == 1 && parts[0] == "findings":
		var sequence *int
		if raw := r.URL.Query().Get("cueSequence"); raw != "" {
			value, parseErr := strconv.Atoi(raw)
			if parseErr != nil {
				h.fail(w, fmt.Errorf("%w: cueSequence 必须是整数", domain.ErrInvalidInput))
				return
			}
			sequence = &value
		}
		out, err = h.app.FilterFindings(projectID, application.FindingFilter{Severity: r.URL.Query().Get("severity"), RuleCode: r.URL.Query().Get("ruleCode"), Status: r.URL.Query().Get("status"), CueSequence: sequence})
	case len(parts) == 1 && parts[0] == "revisions":
		out, err = h.app.ListRevisions(projectID)
	case len(parts) == 2 && parts[0] == "revisions" && parts[1] == "diff":
		out, err = h.app.DiffRevisions(projectID, r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	case len(parts) == 2 && parts[0] == "revisions":
		out, err = h.app.GetRevision(projectID, parts[1])
	case len(parts) == 1 && parts[0] == "glossary":
		version := int64(0)
		if raw := r.URL.Query().Get("version"); raw != "" {
			version, err = strconv.ParseInt(raw, 10, 64)
		}
		if err == nil {
			out, err = h.app.Glossary(projectID, version)
		}
	case len(parts) == 1 && parts[0] == "validation-runs":
		out, err = h.app.ValidationRuns(projectID, r.URL.Query().Get("revisionID"))
	case len(parts) == 2 && parts[0] == "validation-runs" && parts[1] == "compare":
		out, err = h.app.CompareValidationRuns(projectID, r.URL.Query().Get("before"), r.URL.Query().Get("after"))
	case len(parts) == 1 && parts[0] == "review-detail":
		out, err = h.app.ReviewDetail(projectID)
	case len(parts) == 2 && parts[0] == "freeze" && parts[1] == "preview":
		out, err = h.app.FreezePreview(projectID)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) downloadExport(w http.ResponseWriter, projectID, kind string) {
	item, err := h.app.ExportItem(projectID, kind)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", item.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+item.Filename+"\"")
	w.Header().Set("X-Project-ID", item.ProjectID)
	w.Header().Set("X-Manifest-ID", item.ManifestID)
	w.Header().Set("X-Verification-Code", item.VerificationCode)
	w.Header().Set("X-Content-SHA256", item.Digest)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(item.Content)
}

func (h *Handler) handleProjectWrite(w http.ResponseWriter, r *http.Request, projectID string, parts []string) {
	var out any
	var err error
	switch {
	case len(parts) == 1 && parts[0] == "rules":
		var c application.RuleUpdateCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.UpdateProjectRules(projectID, c)
		}
	case len(parts) == 1 && parts[0] == "terms":
		var c application.TermCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.AddTerm(projectID, c)
		}
	case len(parts) == 2 && parts[0] == "terms" && parts[1] == "batch":
		var c application.BatchGlossaryCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.ImportGlossary(projectID, c)
		}
	case len(parts) == 1 && parts[0] == "revisions":
		var c application.RevisionCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.SubmitRevision(projectID, c)
		}
	case len(parts) == 1 && parts[0] == "validate":
		var c application.VersionedCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.Validate(projectID, c)
		}
	case len(parts) == 1 && parts[0] == "submit-review":
		var c application.VersionedCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.SubmitForReview(projectID, c)
		}
	case len(parts) == 1 && parts[0] == "review":
		var c application.ReviewCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.Review(projectID, c)
		}
	case len(parts) == 1 && parts[0] == "freeze":
		var c application.VersionedCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.Freeze(projectID, c)
		}
	case len(parts) == 2 && parts[0] == "findings" && parts[1] == "resolve":
		var c application.ResolveCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.ResolveFinding(projectID, c)
		}
	case len(parts) == 2 && parts[0] == "findings" && parts[1] == "resolve-batch":
		var c application.BatchResolveCommand
		err = decode(r, &c)
		if err == nil {
			out, err = h.app.ResolveFindings(projectID, c)
		}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
