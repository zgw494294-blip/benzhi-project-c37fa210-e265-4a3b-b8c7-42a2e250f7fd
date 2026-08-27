package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"stagecaption-finalizer/internal/application"
	"stagecaption-finalizer/internal/store"
	"stagecaption-finalizer/internal/validation"
	"stagecaption-finalizer/internal/web"
)

func main() {
	addr := flag.String("addr", "", "监听地址")
	selfcheck := flag.Bool("selfcheck", false, "执行完整回环自检")
	dataDir := flag.String("data", ".stagecaption-data", "数据目录")
	flag.Parse()
	listen := *addr
	if listen == "" {
		if port := os.Getenv("PORT"); port != "" {
			listen = "127.0.0.1:" + port
		} else {
			listen = "127.0.0.1:19081"
		}
	}
	if *selfcheck {
		if err := runSelfcheck(listen, *dataDir); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := runServer(listen, *dataDir, nil); err != nil {
		log.Fatal(err)
	}
}

func buildHandler(dataDir string) (http.Handler, error) {
	repo, err := store.Open(dataDir)
	if err != nil {
		return nil, err
	}
	return web.New(application.New(repo, validation.New())).Routes(), nil
}

func runServer(addr, dataDir string, ready chan<- string) error {
	h, err := buildHandler(dataDir)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	ln, err := netListen(addr)
	if err != nil {
		return err
	}
	if ready != nil {
		ready <- "http://" + ln.Addr().String()
	}
	err = srv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func runSelfcheck(addr, dataDir string) error {
	ready := make(chan string, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = runServer(addr, dataDir, ready) }()
	base := <-ready
	client := &http.Client{Timeout: 2 * time.Second}
	post := func(path string, v any) (map[string]any, error) {
		b, _ := json.Marshal(v)
		req, _ := http.NewRequest(http.MethodPost, base+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("%s %d: %s", path, resp.StatusCode, body)
		}
		var out map[string]any
		return out, json.Unmarshal(body, &out)
	}
	p, err := post("/api/projects", map[string]any{"title": "自检演出", "sourceLanguage": "en", "targetLanguage": "zh", "frameRate": 25, "minDisplayMillis": 500, "maxDisplayMillis": 5000, "actor": "负责人", "idempotencyKey": "self-create"})
	if err != nil {
		return err
	}
	pid := p["id"].(string)
	ws, err := getJSON(client, base+"/api/projects/"+pid)
	if err != nil {
		return err
	}
	version := int64(ws["project"].(map[string]any)["version"].(float64))
	if _, err = post("/api/projects/"+pid+"/terms", map[string]any{"sourceText": "Hello", "requiredTranslation": "你好", "actor": "译员", "expectedVersion": version, "idempotencyKey": "self-term"}); err != nil {
		return err
	}
	ws, err = getJSON(client, base+"/api/projects/"+pid)
	if err != nil {
		return err
	}
	version = int64(ws["project"].(map[string]any)["version"].(float64))
	if _, err = post("/api/projects/"+pid+"/revisions", map[string]any{"submittedBy": "译员", "expectedVersion": version, "idempotencyKey": "self-revision", "cues": []map[string]any{{"sequence": 1, "inMillis": 0, "outMillis": 1200, "sourceText": "Hello", "translatedText": "你好"}}}); err != nil {
		return err
	}
	ws, err = getJSON(client, base+"/api/projects/"+pid)
	if err != nil {
		return err
	}
	version = int64(ws["project"].(map[string]any)["version"].(float64))
	if _, err = post("/api/projects/"+pid+"/validate", map[string]any{"actor": "译员", "expectedVersion": version, "idempotencyKey": "self-validate"}); err != nil {
		return err
	}
	ws, err = getJSON(client, base+"/api/projects/"+pid)
	if err != nil {
		return err
	}
	version = int64(ws["project"].(map[string]any)["version"].(float64))
	if _, err = post("/api/projects/"+pid+"/submit-review", map[string]any{"actor": "译员", "expectedVersion": version, "idempotencyKey": "self-submit"}); err != nil {
		return err
	}
	ws, err = getJSON(client, base+"/api/projects/"+pid)
	if err != nil {
		return err
	}
	version = int64(ws["project"].(map[string]any)["version"].(float64))
	if _, err = post("/api/projects/"+pid+"/review", map[string]any{"reviewer": "校审员", "decision": "approve", "expectedVersion": version, "idempotencyKey": "self-review"}); err != nil {
		return err
	}
	ws, err = getJSON(client, base+"/api/projects/"+pid)
	if err != nil {
		return err
	}
	version = int64(ws["project"].(map[string]any)["version"].(float64))
	if _, err = post("/api/projects/"+pid+"/freeze", map[string]any{"actor": "负责人", "expectedVersion": version, "idempotencyKey": "self-freeze"}); err != nil {
		return err
	}
	if _, err = getJSON(client, base+"/api/projects/"+pid+"/verify"); err != nil {
		return err
	}
	if _, err = getJSON(client, base+"/api/projects/"+pid+"/export"); err != nil {
		return err
	}
	_ = ctx
	fmt.Println("自检成功：建档、术语、修订、校验、复核、冻结、导出和核验全部通过")
	return nil
}

func getJSON(c *http.Client, url string) (map[string]any, error) {
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s 返回 %d", url, resp.StatusCode)
	}
	var out map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
}
