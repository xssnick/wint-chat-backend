package matchmaker

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/xssnick/wint/pkg/repo"
)

type Repo interface {
	CreateGame(ctx context.Context, owner uint64, users []uint64) (uint64, error)
	GetGameState(ctx context.Context, user uint64) (repo.MatchState, error)
}

type matchQueue struct {
	wantStart     bool
	searchStarted int64
	owner         playerInfo
	players       map[uint64]*playerInfo
	lockPlayers   sync.RWMutex
}

type playerInfo struct {
	uid   uint64
	seen  int64
	mode  int
	match *matchQueue
}

type Config struct {
	Repo          Repo
	PingValid     int64
	MaxPlayers    int
	MinPlayers    int
	LongSearchSec int64
	TickEvery     time.Duration
}

type Matchmaker struct {
	stopper chan bool

	matches map[uint64]*matchQueue
	players map[uint64]*playerInfo

	lockMatches sync.RWMutex
	lockPlayers sync.RWMutex

	Config
}

func NewMatchmaker(cfg Config) *Matchmaker {
	return &Matchmaker{
		stopper: make(chan bool),
		matches: map[uint64]*matchQueue{},
		players: map[uint64]*playerInfo{},
		Config:  cfg,
	}
}

func (m *Matchmaker) LinkRoutine() {
	mpToDel := make([]uint64, 0, 8)
	mToDel := make([]uint64, 0, 128)
	pToDel := make([]uint64, 0, 1024)

	mToStart := make([]*matchQueue, 0, 32)
	mToFill := [][]*matchQueue{
		make([]*matchQueue, 0, 256),
		make([]*matchQueue, 0, 256),
		make([]*matchQueue, 0, 256),
	}

	log.Println("Matchmaker started!")

	for {
		//log.Println(len(m.matches), len(m.players))
		select {
		case <-time.After(m.TickEvery):
		case <-m.stopper:
			return
		}

		pToDel = pToDel[:0]
		mToDel = mToDel[:0]
		mToStart = mToStart[:0]
		for i := range mToFill {
			mToFill[i] = mToFill[i][:0]
		}

		tm := time.Now().Unix()

		// clean players
		m.lockPlayers.Lock()
		for uid, player := range m.players {
			if player.seen+m.PingValid < tm {
				pToDel = append(pToDel, uid)
			}
		}

		for _, v := range pToDel {
			log.Println("expired player", v)
			delete(m.players, v)
		}
		m.lockPlayers.Unlock()

		// clean matches
		m.lockMatches.Lock()
		for owner, match := range m.matches {
			mpToDel = mpToDel[:0]

			match.lockPlayers.Lock()
			if match.owner.seen+m.PingValid < tm {
				mToDel = append(mToDel, owner)

				for _, p := range match.players {
					p.match = nil
				}
			}

			for uid, p := range match.players {
				if p.seen+m.PingValid < tm {
					mpToDel = append(mpToDel, uid)
				}
			}

			for _, v := range mpToDel {
				delete(match.players, v)
			}
			match.lockPlayers.Unlock()
		}

		for _, v := range mToDel {
			log.Println("expired match", v)
			delete(m.matches, v)
		}

		mToDel = mToDel[:0]

		// start games
		for owner, match := range m.matches {
			if len(match.players) >= m.MaxPlayers || (len(match.players) >= m.MinPlayers && (match.searchStarted+m.LongSearchSec < tm || match.wantStart)) {
				mToDel = append(mToDel, owner)
				mToStart = append(mToStart, match)
			} else {
				if match.owner.mode < len(mToFill) {
					mToFill[match.owner.mode] = append(mToFill[match.owner.mode], match)
				}
			}
		}

		for _, v := range mToDel {
			log.Println("start match", v)
			delete(m.matches, v)
		}
		m.lockMatches.Unlock()

		pToDel = pToDel[:0]

		for _, v := range mToStart {
			v.lockPlayers.RLock()
			for uid := range v.players {
				pToDel = append(pToDel, uid)
			}
			v.lockPlayers.RUnlock()
		}

		for x := range mToFill {
			sort.Slice(mToFill[x], func(i, j int) bool {
				return len(mToFill[x][i].players) < len(mToFill[x][j].players)
			})
		}

		i := 0
		// fill players matches
		m.lockPlayers.Lock()
		for _, v := range pToDel {
			log.Println("start player", v)
			delete(m.players, v)
		}

		for x := range mToFill {
			arr := mToFill[x]

			if len(arr) > 0 {
				for uid, player := range m.players {
					if player.match == nil && player.mode == x {
						randMatch := arr[i%len(arr)]
						i++

						randMatch.lockPlayers.Lock()
						player.match = randMatch
						randMatch.players[uid] = player
						randMatch.lockPlayers.Unlock()
					}
				}
			}
		}
		m.lockPlayers.Unlock()

		for _, v := range mToStart {
			v.lockPlayers.RLock()
			var players = make([]uint64, 0, len(v.players))
			for uid := range v.players {
				players = append(players, uid)
			}
			v.lockPlayers.RUnlock()

			log.Println("game created, owner", v)

			_, err := m.Repo.CreateGame(context.Background(), v.owner.uid, players)
			if err != nil {
				log.Println("[Maker Routine] create match error:", err)
			}

			m.lockMatches.Lock()
			delete(m.matches, v.owner.uid)
			m.lockMatches.Unlock()

			m.lockPlayers.Lock()
			for _, p := range players {
				delete(m.players, p)
			}
			m.lockPlayers.Unlock()
		}
	}
}

func (m *Matchmaker) FindMatch(user uint64, mode int) error {
	if state, err := m.StateMatch(user, false); err == repo.ErrNotFound || (err == nil && state.FinishedAt > 0) {
		m.lockPlayers.Lock()
		m.players[user] = &playerInfo{
			seen:  time.Now().Unix(),
			mode:  mode,
			match: nil,
		}
		m.lockPlayers.Unlock()

		log.Println("FIND")
		return nil
	}

	return repo.ErrAlreadyInSearch
}

func (m *Matchmaker) CreateMatch(user uint64, mode int) error {
	if state, err := m.StateMatch(user, false); err == repo.ErrNotFound || (err == nil && state.FinishedAt > 0) {
		m.lockMatches.Lock()
		m.matches[user] = &matchQueue{
			searchStarted: time.Now().Unix(),
			owner: playerInfo{
				uid:  user,
				seen: time.Now().Unix(),
				mode: mode,
			},
			players: map[uint64]*playerInfo{},
		}
		m.lockMatches.Unlock()

		return nil
	}

	return repo.ErrAlreadyInSearch
}

func (m *Matchmaker) StartMatch(user uint64) error {
	m.lockMatches.RLock()
	defer m.lockMatches.RUnlock()

	match := m.matches[user]
	if match != nil {
		if len(match.players) >= m.MinPlayers {
			match.wantStart = true
			return nil
		}
		return repo.ErrNotEnoughPlayers
	}

	return repo.ErrNotFound
}

func (m *Matchmaker) StateMatch(user uint64, withPing bool) (repo.MatchState, error) {
	m.lockMatches.RLock()
	match := m.matches[user]
	m.lockMatches.RUnlock()

	if match != nil {
		players := make([]uint64, 0, len(match.players))
		log.Println("check state", user, len(match.players))

		match.lockPlayers.RLock()
		for p := range match.players {
			players = append(players, p)
		}
		match.lockPlayers.RUnlock()

		if withPing {
			match.owner.seen = time.Now().Unix()
		}

		return repo.MatchState{
			Players: players,
		}, nil
	}

	m.lockPlayers.RLock()
	player := m.players[user]
	m.lockPlayers.RUnlock()

	if player != nil {
		if withPing {
			player.seen = time.Now().Unix()
		}

		if player.match == nil {
			return repo.MatchState{
				Players: []uint64{},
			}, nil
		}

		players := make([]uint64, 0, len(player.match.players))

		player.match.lockPlayers.RLock()
		for p := range player.match.players {
			players = append(players, p)
		}
		player.match.lockPlayers.RUnlock()

		return repo.MatchState{
			Players: players,
		}, nil
	}

	return m.Repo.GetGameState(context.Background(), user)
}

func (m *Matchmaker) ExitMatch(user uint64) error {
	m.lockMatches.RLock()
	match := m.matches[user]
	m.lockMatches.RUnlock()

	if match != nil {
		m.lockMatches.Lock()
		delete(m.matches, user)
		m.lockMatches.Unlock()

		match.lockPlayers.RLock()
		for _, p := range match.players {
			p.match = nil
		}
		match.lockPlayers.RUnlock()
		return nil
	}

	m.lockPlayers.RLock()
	player := m.players[user]
	m.lockPlayers.RUnlock()

	if player != nil {
		if player.match != nil {
			m.lockPlayers.Lock()
			delete(m.players, user)
			m.lockPlayers.Unlock()

			player.match.lockPlayers.Lock()
			delete(player.match.players, user)
			player.match.lockPlayers.Unlock()
		} else {
			m.lockPlayers.Lock()
			delete(m.players, user)
			m.lockPlayers.Unlock()
		}
		return nil
	}

	return repo.ErrNotFound
}
