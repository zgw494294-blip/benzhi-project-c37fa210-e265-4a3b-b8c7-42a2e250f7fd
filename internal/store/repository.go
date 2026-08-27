package store

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"
)

type Repository struct {
	dir          string
	mu           sync.RWMutex
	projectLocks sync.Map
	state        Snapshot
	events       []AuditEvent
	head         string
	now          func() time.Time
}

func Open(dir string) (*Repository, error) {
	r := &Repository{dir: dir, state: emptySnapshot(), now: time.Now}
	if err := r.recover(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Repository) lockFor(projectID string) *sync.Mutex {
	v, _ := r.projectLocks.LoadOrStore(projectID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func cloneSnapshot(s Snapshot) (Snapshot, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return Snapshot{}, err
	}
	var out Snapshot
	if err = json.Unmarshal(b, &out); err != nil {
		return Snapshot{}, err
	}
	return out, nil
}

func (r *Repository) Read(fn func(Snapshot) error) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copyState, err := cloneSnapshot(r.state)
	if err != nil {
		return err
	}
	return fn(copyState)
}

func (r *Repository) Update(projectID, eventType, actor string, fn func(*Snapshot) error) error {
	lock := r.lockFor(projectID)
	lock.Lock()
	defer lock.Unlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	next, err := cloneSnapshot(r.state)
	if err != nil {
		return err
	}
	ensureSnapshot(&next)
	if err = fn(&next); err != nil {
		return err
	}
	if reflect.DeepEqual(next, r.state) {
		return nil
	}
	projection, err := json.Marshal(next)
	if err != nil {
		return err
	}
	event := AuditEvent{Sequence: int64(len(r.events) + 1), ProjectID: projectID, Type: eventType, Actor: actor, OccurredAt: r.now().UTC(), Previous: r.head, Projection: projection}
	event.Digest = eventDigest(event)
	if err = r.appendEvent(event); err != nil {
		return fmt.Errorf("追加审计事件: %w", err)
	}
	if err = r.writeProjection(next); err != nil {
		return fmt.Errorf("写入投影: %w", err)
	}
	r.state = next
	r.events = append(r.events, event)
	r.head = event.Digest
	return nil
}

func (r *Repository) AuditHead() string { r.mu.RLock(); defer r.mu.RUnlock(); return r.head }

func (r *Repository) Events(projectID string) []EventView {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]EventView, 0)
	for _, e := range r.events {
		if projectID == "" || e.ProjectID == projectID {
			out = append(out, EventView{e.Sequence, e.ProjectID, e.Type, e.Actor, e.OccurredAt, e.Previous, e.Digest})
		}
	}
	return out
}

func (r *Repository) VerifyAudit() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	events, err := r.readPersistedEvents()
	if err != nil {
		return err
	}
	return verifyEvents(events)
}

func (r *Repository) VerifyAuditAnchor(digest string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	events, err := r.readPersistedEvents()
	if err != nil || verifyEvents(events) != nil {
		return false
	}
	if digest == "" {
		return len(events) == 0
	}
	for _, event := range events {
		if event.Digest == digest {
			return true
		}
	}
	return false
}

func (r *Repository) VerifyProjection() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, err := os.ReadFile(r.projectionPath())
	if err != nil {
		return err
	}
	current, err := json.Marshal(r.state)
	if err != nil {
		return err
	}
	var persistedValue, currentValue any
	if err = json.Unmarshal(b, &persistedValue); err != nil {
		return err
	}
	if err = json.Unmarshal(current, &currentValue); err != nil {
		return err
	}
	if !reflect.DeepEqual(persistedValue, currentValue) {
		return fmt.Errorf("持久化投影与当前投影不一致")
	}
	return nil
}

func verifyEvents(events []AuditEvent) error {
	previous := ""
	for i, e := range events {
		if e.Sequence != int64(i+1) || e.Previous != previous {
			return fmt.Errorf("事件 %d 前向摘要不匹配", e.Sequence)
		}
		if eventDigest(e) != e.Digest {
			return fmt.Errorf("事件 %d 摘要不匹配", e.Sequence)
		}
		previous = e.Digest
	}
	return nil
}
