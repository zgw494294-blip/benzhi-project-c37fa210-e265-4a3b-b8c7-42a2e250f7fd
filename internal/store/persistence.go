package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (r *Repository) eventPath() string      { return filepath.Join(r.dir, "audit.jsonl") }
func (r *Repository) projectionPath() string { return filepath.Join(r.dir, "projection.json") }

func eventDigest(e AuditEvent) string {
	copyEvent := e
	copyEvent.Digest = ""
	b, _ := json.Marshal(copyEvent)
	return digestBytes(b)
}

func digestBytes(b []byte) string {
	return fmt.Sprintf("%x", sha256sum(b))
}

func sha256sum(b []byte) [32]byte { return sha256Sum(b) }

func (r *Repository) appendEvent(event AuditEvent) error {
	if err := os.MkdirAll(r.dir, 0750); err != nil {
		return err
	}
	f, err := os.OpenFile(r.eventPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	if err = enc.Encode(event); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func (r *Repository) writeProjection(state Snapshot) error {
	if err := os.MkdirAll(r.dir, 0750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(r.dir, "projection-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			os.Remove(name)
		}
	}()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err = enc.Encode(state); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, r.projectionPath()); err != nil {
		return err
	}
	ok = true
	dir, err := os.Open(r.dir)
	if err == nil {
		err = dir.Sync()
		dir.Close()
	}
	return err
}

func (r *Repository) recover() error {
	if err := os.MkdirAll(r.dir, 0750); err != nil {
		return err
	}
	f, err := os.Open(r.eventPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	previous := ""
	var last Snapshot
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			var e AuditEvent
			if err = json.Unmarshal(line, &e); err != nil {
				return fmt.Errorf("审计日志损坏: %w", err)
			}
			if e.Sequence != int64(len(r.events)+1) || e.Previous != previous || eventDigest(e) != e.Digest {
				return fmt.Errorf("审计摘要链在序号 %d 损坏", e.Sequence)
			}
			if err = json.Unmarshal(e.Projection, &last); err != nil {
				return fmt.Errorf("事件投影损坏: %w", err)
			}
			r.events = append(r.events, e)
			previous = e.Digest
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if len(r.events) > 0 {
		ensureSnapshot(&last)
		r.state = last
		r.head = previous
		if err = r.writeProjection(last); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) readPersistedEvents() ([]AuditEvent, error) {
	f, err := os.Open(r.eventPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	events := make([]AuditEvent, 0)
	decoder := json.NewDecoder(f)
	for {
		var event AuditEvent
		if err = decoder.Decode(&event); err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("审计日志损坏: %w", err)
		}
		events = append(events, event)
	}
	return events, nil
}
