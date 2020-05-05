package gaming

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/xssnick/wint/pkg/repo"
)

type Repo interface {
	GetMessagesGame(ctx context.Context, gameid uint64) ([]repo.Message, error)
	GetActiveGameIDs(ctx context.Context) ([]uint64, error)
	GetGameStateByID(ctx context.Context, gameid, uid uint64, unsafe bool) (repo.MatchState, error)
	KickFromGame(ctx context.Context, owner, game, user uint64) error
	FinishGame(ctx context.Context, owner, game, winner uint64) error
}

type Poller interface {
	PushEvent(id uint64)
}

type Gaming struct {
	poller  Poller
	repo    Repo
	every   time.Duration
	stopper chan bool
	wg      sync.WaitGroup
}

func (g *Gaming) CanWriteLeft(uid uint64, mstate *repo.MatchState, msgs []repo.Message) (bool, bool, bool, int64) {
	var roundStartTime = mstate.StartedAt
	var wrotePlayers int
	var canWrite = mstate.Owner == uid
	var myTurn = mstate.Owner == uid
	var canDelete = false

	for _, v := range msgs {
		canDelete = false

		if mstate.Owner != v.Owner {
			wrotePlayers++

			if wrotePlayers >= len(mstate.Players) {
				// owner turn
				roundStartTime = v.CreatedAt
				myTurn = mstate.Owner == uid

				if wrotePlayers == len(mstate.Players) {
					canDelete = true
				}
			}
		} else {
			myTurn = mstate.Owner != uid
			roundStartTime = v.CreatedAt
		}

		if mstate.Owner == uid {
			if v.Owner == mstate.Owner {
				canWrite = false
				wrotePlayers = 0
			} else {
				if wrotePlayers >= len(mstate.Players) {
					canWrite = true
				}
			}
		} else {
			if v.Owner == mstate.Owner {
				canWrite = true
				wrotePlayers = 0
			} else if v.Owner == uid {
				canWrite = false
			}
		}
	}

	if time.Now().Unix()-roundStartTime > 120 {
		canWrite = false
	}

	return canWrite && !canDelete, myTurn, canDelete, roundStartTime
}

func (g *Gaming) WhoFuckedUp(mstate *repo.MatchState, msgs []repo.Message) []uint64 {
	var roundStartTime = mstate.StartedAt

	players := map[uint64]bool{}
	ownerAnswered := false

	for _, v := range msgs {
		if mstate.Owner != v.Owner {
			players[v.Owner] = true

			if len(players) == len(mstate.Players) {
				// owner turn
				roundStartTime = v.CreatedAt
				ownerAnswered = false
			}
		} else {
			roundStartTime = v.CreatedAt
			players = map[uint64]bool{}
			ownerAnswered = true
		}
	}

	if roundStartTime+120 >= time.Now().Unix() {
		return []uint64{}
	}

	if !ownerAnswered {
		return []uint64{mstate.Owner}
	}

	arr := make([]uint64, 0, len(players))
	for _, p := range mstate.Players {
		if !players[p] {
			arr = append(arr, p)
		}
	}

	return arr
}

func (g *Gaming) StartCheckGames() {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()

		for {
			select {
			case <-g.stopper:
				return
			case <-time.After(g.every):
			}

			ids, err := g.repo.GetActiveGameIDs(context.Background())
			if err != nil {
				log.Println("CHECK GAMES ERR:", err)
				continue
			}

		iterGames:
			for _, id := range ids {
				mstate, err := g.repo.GetGameStateByID(context.Background(), id, 0, true)
				if err != nil {
					log.Println("CHECK GAME GET STATE ERR:", id, err)
					continue
				}

				msgs, err := g.repo.GetMessagesGame(context.Background(), id)
				if err != nil {
					log.Println("CHECK GAME GET MESSAGES ERR:", id, err)
					continue
				}

				/*var last uint64
				if len(msgs) > 0 {
					last = msgs[len(msgs)-1].ID
				}*/

				pls := g.WhoFuckedUp(&mstate, msgs)
				for _, p := range pls {
					if p == mstate.Owner {
						err = g.repo.FinishGame(context.Background(), mstate.Owner, id, 0)
						if err != nil {
							log.Println("CHECK GAME CHECK PLAYERS FINISH ERR:", id, p, err)
							continue
						}

						g.poller.PushEvent(id)

						continue iterGames
					}

					err = g.repo.KickFromGame(context.Background(), mstate.Owner, id, p)
					if err != nil {
						log.Println("CHECK GAME CHECK PLAYERS KICK ERR:", id, p, err)
						continue
					}
				}

				pleft := len(mstate.Players) - len(pls)
				if pleft <= 1 {
					var winner uint64
					if pleft == 1 {
						for _, x := range mstate.Players {
							found := false
							for _, y := range pls {
								if x == y {
									found = true
									break
								}
							}

							if !found {
								winner = x
								break
							}
						}
					}

					err = g.repo.FinishGame(context.Background(), mstate.Owner, id, winner)
					if err != nil {
						log.Println("CHECK GAME LEFT PLAYERS FINISH ERR:", id, winner, err)
						continue
					}
				}

				if len(pls) > 0 || pleft <= 1 {
					g.poller.PushEvent(id)
				}
			}
		}
	}()
}

func (g *Gaming) Stop() {
	g.stopper <- true
	g.wg.Wait()
}

func NewGaming(every time.Duration, repo Repo, poller Poller) *Gaming {
	return &Gaming{
		poller:  poller,
		repo:    repo,
		every:   every,
		stopper: make(chan bool, 1),
	}
}
