package storage

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrNoSAToken = errors.New("no available service account token")

type SAEntry struct {
	ID     string
	Token  string
	Weight int
	MaxQPS int
}

type saState struct {
	entry          SAEntry
	usedInWindow   int
	windowStart    time.Time
	consecutiveErr int
	openUntil      time.Time
	halfOpen       bool
}

type SAPool struct {
	mu       sync.Mutex
	clock    func() time.Time
	states   []*saState
	schedule []*saState
	nextIdx  int
}

func NewSAPool(entries []SAEntry) (*SAPool, error) {
	if len(entries) == 0 {
		return nil, errors.New("empty sa entries")
	}
	states := make([]*saState, 0, len(entries))
	for _, e := range entries {
		if e.Weight <= 0 {
			e.Weight = 1
		}
		if e.MaxQPS <= 0 {
			e.MaxQPS = 1
		}
		states = append(states, &saState{entry: e})
	}

	pool := &SAPool{
		clock:  time.Now,
		states: states,
	}
	pool.rebuildSchedule()
	return pool, nil
}

func (p *SAPool) Acquire() (SAEntry, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.clock()
	for i := 0; i < len(p.schedule); i++ {
		idx := (p.nextIdx + i) % len(p.schedule)
		state := p.schedule[idx]
		if !state.openUntil.IsZero() && now.Before(state.openUntil) {
			continue
		}
		if !state.openUntil.IsZero() && now.After(state.openUntil) {
			state.openUntil = time.Time{}
			state.halfOpen = true
		}

		if now.Sub(state.windowStart) >= time.Second {
			state.windowStart = now
			state.usedInWindow = 0
		}
		if state.usedInWindow >= state.entry.MaxQPS {
			continue
		}

		state.usedInWindow++
		p.nextIdx = idx + 1
		return state.entry, nil
	}

	return SAEntry{}, ErrNoSAToken
}

func (p *SAPool) ReportResult(saID string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	state := p.find(saID)
	if state == nil {
		return
	}

	if err == nil {
		state.consecutiveErr = 0
		state.halfOpen = false
		state.openUntil = time.Time{}
		return
	}

	state.consecutiveErr++
	if state.halfOpen {
		state.openUntil = p.clock().Add(2 * time.Second)
		state.halfOpen = false
		return
	}

	if state.consecutiveErr >= 3 {
		state.openUntil = p.clock().Add(2 * time.Second)
		state.consecutiveErr = 0
	}
}

func (p *SAPool) find(id string) *saState {
	for _, st := range p.states {
		if st.entry.ID == id {
			return st
		}
	}
	return nil
}

func (p *SAPool) rebuildSchedule() {
	sort.Slice(p.states, func(i, j int) bool {
		return p.states[i].entry.ID < p.states[j].entry.ID
	})
	schedule := make([]*saState, 0)
	for _, st := range p.states {
		for i := 0; i < st.entry.Weight; i++ {
			schedule = append(schedule, st)
		}
	}
	p.schedule = schedule
}
