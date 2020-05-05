package typing

import (
	"sync"
	"time"
)

type typing struct {
	user uint64
	tm time.Time
}

type Typer struct {
	poller Poller
	users   map[uint64][]typing
	lock    sync.RWMutex
	stopper chan bool
}

type Poller interface {
	PushEvent(id uint64)
}

func (t *Typer) Cleaner() {
	for {
		select {
		case <-t.stopper:
			return
		case <-time.After(1 * time.Second):
		}

		now := time.Now()

		mp := map[uint64][]typing{}
		psh := map[uint64]bool{}

		t.lock.RLock()
		for id, tp := range t.users {
			tps := make([]typing,0,len(tp))
			for _, v := range tp {
				if now.Sub(v.tm).Seconds() <= 3 {
					tps = append(tps, v)
				} else {
					psh[id] = true
				}
			}

			if len(tps) > 0 {
				mp[id] = tps
			}
		}
		t.lock.RUnlock()

		for key := range psh {
			t.poller.PushEvent(key)
		}

		t.lock.Lock()
		t.users = mp
		t.lock.Unlock()
	}
}

func (t *Typer) SetType(on, user uint64) {
	t.lock.RLock()
	u := t.users[on]
	t.lock.RUnlock()

	if u == nil {
		u = []typing{{
			user: user,
			tm:   time.Now(),
		}}

		t.lock.Lock()
		t.users[on] = u
		t.lock.Unlock()

		t.poller.PushEvent(on)

		return
	}

	for i := range u {
		if u[i].user == user {
			u[i].tm = time.Now()

			return
		}
	}

	u = append(u, typing{
		user: user,
		tm:   time.Now(),
	})

	t.lock.Lock()
	t.users[on] = u
	t.lock.Unlock()

	t.poller.PushEvent(on)
}

func (t *Typer) DelType(on, user uint64) {
	t.lock.RLock()
	u := t.users[on]
	t.lock.RUnlock()

	if u == nil {
		return
	}

	for i := range u {
		if u[i].user == user {
			t.lock.Lock()
			t.users[on] = append(u[:i], u[i+1:]...)
			t.lock.Unlock()

			t.poller.PushEvent(on)

			return
		}
	}
}

func (t *Typer) GetTypes(on uint64, users []uint64) []uint64 {
	res := make([]uint64, 0, len(users))

	t.lock.RLock()
	tps := t.users[on]
	t.lock.RUnlock()

	for _, u := range users {
		for _, t := range tps {
			if u == t.user {
				res = append(res, u)
				break
			}
		}
	}

	return res
}

func NewTyper(poller Poller) *Typer {
	return &Typer{
		poller:poller,
		users:   map[uint64][]typing{},
		stopper: make(chan bool),
	}
}
