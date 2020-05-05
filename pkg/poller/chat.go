package poller

import (
	"log"
	"strconv"
	"sync"
	"time"
)

type waiter struct {
	ch      chan bool
	created time.Time
}

type room struct {
	event     uint64
	chans    []waiter
	created  time.Time
	accessed time.Time
	sync.RWMutex
}

type Poller struct {
	rooms   map[uint64]*room
	lock    sync.RWMutex
	stopper chan bool
	wg      sync.WaitGroup
}

func (p *Poller) StartCleaner(tick time.Duration) {
	p.wg.Add(1)
	go func() {
		for {
			select {
			case <-p.stopper:
				p.wg.Done()
				return
			case <-time.After(tick):
			}

			nw := time.Now()
			old := make([]uint64, 0, len(p.rooms)/10)

			p.lock.RLock()
			for ri, r := range p.rooms {
				if nw.Sub(r.accessed) > 1*time.Minute {
					old = append(old, ri)
				} else {
					vchans := make([]waiter, 0, len(r.chans))
					r.Lock()
					for i := range r.chans {
						if nw.Sub(r.chans[i].created) <= 10*time.Second {
							vchans = append(vchans, r.chans[i])
						}
					}
					r.chans = vchans
					r.Unlock()
				}
			}
			p.lock.RUnlock()

			if len(old) > 0 {
				p.lock.Lock()
				for _, v := range old {
					delete(p.rooms, v)
				}
				p.lock.Unlock()
			}
		}
	}()
}

func (p *Poller) StopCleaner() {
	p.stopper <- true
	p.wg.Wait()
}

func (p *Poller) PushEvent(id uint64) {
	log.Println("wanter",id)

	p.lock.RLock()
	r := p.rooms[id]
	p.lock.RUnlock()

	if r == nil {
		p.lock.Lock()
		r = p.rooms[id]
		if r == nil {
			r = &room{
				event:     0,
				chans:    nil,
				created:  time.Now(),
				accessed: time.Now(),
			}

			log.Println("EVENT",r.event)

			p.rooms[id] = r
		}
		p.lock.Unlock()

		return
	}

	r.Lock()
	for _, c := range r.chans {
		close(c.ch)
	}
	r.Unlock()

	r = &room{
		event:     r.event+1,
		chans:    nil,
		created:  time.Now(),
		accessed: time.Now(),
	}

	p.lock.Lock()
	p.rooms[id] = r
	p.lock.Unlock()

	log.Println("EVENT", r.event)
}

func (p *Poller) GetEvent(id uint64) string {
	p.lock.RLock()
	r := p.rooms[id]
	p.lock.RUnlock()

	if r == nil {
		return ""
	}

	return strconv.FormatUint(r.event,10)
}

func (p *Poller) WaitEvent(e uint64, id uint64) (chan bool, bool) {
	p.lock.RLock()
	r := p.rooms[id]
	p.lock.RUnlock()

	ch := make(chan bool, 1)

	found := r != nil

	if found && r.event == e {
		r.Lock()
		r.accessed = time.Now()
		r.chans = append(r.chans, waiter{
			ch:      ch,
			created: time.Now(),
		})
		r.Unlock()

		return ch, true
	}

	close(ch)

	return ch, found
}

func (p *Poller) ForgotEvent(id uint64, ch chan bool) {
	p.lock.RLock()
	r := p.rooms[id]
	p.lock.RUnlock()

	if r == nil {
		return
	}

	r.Lock()
	defer r.Unlock()

	for i := range r.chans {
		if r.chans[i].ch == ch {
			r.chans = append(r.chans[:i], r.chans[i+1:]...)
			return
		}
	}
}

func NewPoller() *Poller {
	return &Poller{
		rooms: make(map[uint64]*room, 1500),
	}
}
