package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"stagecaption-finalizer/internal/application"
	"stagecaption-finalizer/internal/store"
	"stagecaption-finalizer/internal/validation"
)

func TestWorkbenchServesInteractiveFrontendAssets(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New(application.New(repo, validation.New())).Routes())
	defer server.Close()

	tests := []struct {
		path        string
		contentType string
		evidence    []string
	}{
		{path: "/workbench", contentType: "text/html", evidence: []string{"id=\"create\"", "id=\"submitRevision\"", "id=\"freeze\"", "/static/app.js"}},
		{path: "/static/app.css", contentType: "text/css", evidence: []string{".panel", "@media", ".downloads"}},
		{path: "/static/app.js", contentType: "application/javascript", evidence: []string{"fetch(path", "loadWorkspace", "/submit-review", "/freeze/preview"}},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			response, requestErr := server.Client().Get(server.URL + test.path)
			if requestErr != nil {
				t.Fatal(requestErr)
			}
			defer response.Body.Close()
			body, readErr := io.ReadAll(response.Body)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if response.StatusCode != http.StatusOK {
				t.Fatalf("%s 返回 %d", test.path, response.StatusCode)
			}
			if !strings.HasPrefix(response.Header.Get("Content-Type"), test.contentType) {
				t.Fatalf("%s Content-Type=%q", test.path, response.Header.Get("Content-Type"))
			}
			for _, evidence := range test.evidence {
				if !bytes.Contains(body, []byte(evidence)) {
					t.Errorf("%s 缺少前端证据 %q", test.path, evidence)
				}
			}
		})
	}
}

func requestJSON(t *testing.T, client *http.Client, method, url string, body any, status int) map[string]any {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != status {
		t.Fatalf("%s %s 返回 %d，正文 %s", method, url, resp.StatusCode, data)
	}
	var out map[string]any
	if len(data) > 0 {
		if err = json.Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestExtendedRoutesReachCompleteWorkflow(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New(application.New(repo, validation.New())).Routes())
	defer server.Close()
	client := server.Client()
	p := requestJSON(t, client, http.MethodPost, server.URL+"/api/projects", map[string]any{"title": "巡演", "sourceLanguage": "en", "targetLanguage": "zh", "frameRate": 25, "minDisplayMillis": 500, "maxDisplayMillis": 5000, "actor": "负责人", "idempotencyKey": "create"}, http.StatusCreated)
	id, version := p["id"].(string), int64(p["version"].(float64))
	p = requestJSON(t, client, http.MethodPut, server.URL+"/api/projects/"+id+"/rules", map[string]any{"title": "巡演定稿", "sourceLanguage": "en", "targetLanguage": "zh", "frameRate": 25, "minDisplayMillis": 500, "maxDisplayMillis": 4500, "expectedVersion": version, "actor": "负责人", "idempotencyKey": "rules"}, http.StatusOK)
	version = int64(p["version"].(float64))
	batch := requestJSON(t, client, http.MethodPost, server.URL+"/api/projects/"+id+"/terms/batch", map[string]any{"entries": []map[string]any{{"sourceText": "King", "requiredTranslation": "王"}, {"sourceText": "Queen", "requiredTranslation": "王后"}}, "expectedVersion": version, "actor": "译员", "idempotencyKey": "terms"}, http.StatusOK)
	if batch["importedCount"].(float64) != 2 {
		t.Fatal("批量术语路由未导入全部条目")
	}
	workspace := requestJSON(t, client, http.MethodGet, server.URL+"/api/projects/"+id, nil, http.StatusOK)
	version = int64(workspace["project"].(map[string]any)["version"].(float64))
	r1 := requestJSON(t, client, http.MethodPost, server.URL+"/api/projects/"+id+"/revisions", map[string]any{"submittedBy": "译员", "summary": "初稿", "expectedVersion": version, "idempotencyKey": "r1", "cues": []map[string]any{{"sequence": 1, "inMillis": 0, "outMillis": 1000, "sourceText": "King", "translatedText": "王"}}}, http.StatusOK)
	workspace = requestJSON(t, client, http.MethodGet, server.URL+"/api/projects/"+id, nil, http.StatusOK)
	version = int64(workspace["project"].(map[string]any)["version"].(float64))
	r2 := requestJSON(t, client, http.MethodPost, server.URL+"/api/projects/"+id+"/revisions", map[string]any{"submittedBy": "译员", "summary": "补充", "expectedVersion": version, "idempotencyKey": "r2", "cues": []map[string]any{{"sequence": 1, "inMillis": 0, "outMillis": 1000, "sourceText": "King", "translatedText": "王"}, {"sequence": 2, "inMillis": 1100, "outMillis": 1800, "sourceText": "Queen", "translatedText": "王后"}}}, http.StatusOK)
	diffURL := fmt.Sprintf("%s/api/projects/%s/revisions/diff?from=%s&to=%s", server.URL, id, r1["id"], r2["id"])
	diff := requestJSON(t, client, http.MethodGet, diffURL, nil, http.StatusOK)
	if len(diff["changes"].([]any)) != 1 {
		t.Fatal("修订差异路由结果错误")
	}
	workspace = requestJSON(t, client, http.MethodGet, server.URL+"/api/projects/"+id, nil, http.StatusOK)
	version = int64(workspace["project"].(map[string]any)["version"].(float64))
	requestJSON(t, client, http.MethodPost, server.URL+"/api/projects/"+id+"/validate", map[string]any{"actor": "译员", "expectedVersion": version, "idempotencyKey": "validate"}, http.StatusOK)
	runsResponse, err := client.Get(server.URL + "/api/projects/" + id + "/validation-runs?revisionID=" + r2["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	var runs []map[string]any
	if runsResponse.StatusCode != http.StatusOK || json.NewDecoder(runsResponse.Body).Decode(&runs) != nil || len(runs) != 1 {
		_ = runsResponse.Body.Close()
		t.Fatal("校验运行查询路由结果错误")
	}
	_ = runsResponse.Body.Close()
	workspace = requestJSON(t, client, http.MethodGet, server.URL+"/api/projects/"+id, nil, http.StatusOK)
	version = int64(workspace["project"].(map[string]any)["version"].(float64))
	requestJSON(t, client, http.MethodPost, server.URL+"/api/projects/"+id+"/submit-review", map[string]any{"actor": "译员", "expectedVersion": version, "idempotencyKey": "submit"}, http.StatusOK)
	workspace = requestJSON(t, client, http.MethodGet, server.URL+"/api/projects/"+id, nil, http.StatusOK)
	version = int64(workspace["project"].(map[string]any)["version"].(float64))
	requestJSON(t, client, http.MethodPost, server.URL+"/api/projects/"+id+"/review", map[string]any{"reviewer": "校审", "decision": "approve", "expectedVersion": version, "idempotencyKey": "review"}, http.StatusOK)
	requestJSON(t, client, http.MethodGet, server.URL+"/api/projects/"+id+"/review-detail", nil, http.StatusOK)
	preview := requestJSON(t, client, http.MethodGet, server.URL+"/api/projects/"+id+"/freeze/preview", nil, http.StatusOK)
	workspace = requestJSON(t, client, http.MethodGet, server.URL+"/api/projects/"+id, nil, http.StatusOK)
	version = int64(workspace["project"].(map[string]any)["version"].(float64))
	manifest := requestJSON(t, client, http.MethodPost, server.URL+"/api/projects/"+id+"/freeze", map[string]any{"actor": "负责人", "expectedVersion": version, "idempotencyKey": "freeze"}, http.StatusOK)
	if preview["captionDigest"] != manifest["captionDigest"] {
		t.Fatal("冻结预览与清单摘要不同")
	}
	for _, kind := range []string{"captions", "glossary", "audit"} {
		resp, requestErr := client.Get(server.URL + "/api/projects/" + id + "/exports/" + kind)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || resp.Header.Get("X-Manifest-ID") != manifest["id"] || resp.Header.Get("X-Content-SHA256") == "" {
			t.Fatalf("%s 分项下载元数据错误", kind)
		}
	}
	verified := requestJSON(t, client, http.MethodGet, server.URL+"/api/projects/"+id+"/verify", nil, http.StatusOK)
	if verified["valid"] != true {
		t.Fatalf("HTTP 核验失败: %#v", verified)
	}
}
