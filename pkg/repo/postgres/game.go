package postgres

import (
	"context"
	"database/sql"
	"log"
	"sort"
	"time"

	"github.com/xssnick/wint/pkg/repo"
)

func (r *Repo) GetActiveGameIDs(ctx context.Context) ([]uint64, error) {
	var games []uint64

	err := r.db.SelectContext(ctx, &games, `SELECT id FROM games WHERE finished_at IS NULL`)
	if err != nil {
		if err == sql.ErrNoRows {
			return []uint64{}, nil
		}

		return nil, err
	}

	return games, nil
}

func (r *Repo) GetMessagesGame(ctx context.Context, gameid uint64) ([]repo.Message, error) {
	var msgs []struct {
		ID        uint64    `db:"id"`
		Name      string    `json:"name"`
		Owner     uint64    `db:"owner"`
		Text      string    `db:"msg"`
		CreatedAt time.Time `db:"created_at"`
	}

	err := r.db.SelectContext(ctx, &msgs, `SELECT m.id as id, m.created_at as created_at, m.owner as owner, m.msg as msg, u.name as name FROM messages m JOIN users u ON u.id=m.owner WHERE m.game = $1 ORDER BY m.created_at ASC`, gameid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}

		return nil, err
	}

	result := make([]repo.Message, len(msgs))
	for i, v := range msgs {
		result[i] = repo.Message{
			ID:        v.ID,
			Name:      v.Name,
			Owner:     v.Owner,
			Text:      v.Text,
			CreatedAt: v.CreatedAt.Unix(),
		}
	}

	return result, nil
}

func (r *Repo) GetLikesGame(ctx context.Context, gameid uint64) ([]repo.Like, error) {
	var likes []struct {
		MessageID uint64 `db:"message"`
		UserID    uint64 `db:"owner"`
	}

	err := r.db.SelectContext(ctx, &likes, `SELECT l.message as message, l.owner as owner FROM likes l JOIN messages m ON m.id=l.message JOIN games g ON g.id=m.game WHERE g.id = $1`, gameid)
	if err != nil {
		if err == sql.ErrNoRows {
			return []repo.Like{}, nil
		}

		return nil, err
	}

	result := make([]repo.Like, len(likes))
	for i, v := range likes {
		result[i] = repo.Like{
			MessageID: v.MessageID,
			UserID:    v.UserID,
		}
	}

	return result, nil
}

func (r *Repo) KickFromGame(ctx context.Context, owner, game, user uint64) error {
	log.Println("KICKING:", game, user)

	_, err := r.db.ExecContext(ctx, "UPDATE plays SET lost_at=$1 FROM games g WHERE player=$2 AND game=$3 AND g.id=game AND g.owner=$4", time.Now().UTC(), user, game, owner)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) FinishGame(ctx context.Context, owner, game, winner uint64) error {
	log.Println("FINISH:", game, winner)

	var winptr *uint64
	if winner > 0 {
		winptr = &winner
	}

	_, err := r.db.ExecContext(ctx, "UPDATE games SET winner=$1, finished_at=$2 WHERE owner=$3 AND id=$4", winptr, time.Now().UTC(), owner, game)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) GetGames(ctx context.Context, uid uint64) ([]*repo.Game, error) {
	var plays []struct {
		ID         uint64    `db:"id"`
		UID        uint64    `db:"uid"`
		Winner     *uint64   `db:"winner"`
		Owner     uint64   `db:"owner"`
		Name       string    `db:"name"`
		FinishedAt time.Time `db:"finished_at"`
	}

	err := r.db.SelectContext(ctx, &plays, `SELECT DISTINCT ON (u.id,g.id) u.id as uid, u.name as name,g.id as id,g.finished_at as finished_at, g.winner as winner, g.owner as owner from users u join plays p on p.player=u.id JOIN games g on p.game = g.id or g.owner=u.id WHERE g.id
    IN (SELECT gm.id from games gm JOIN plays pl ON pl.game=gm.id WHERE gm.finished_at IS NOT NULL AND (pl.player=$1 OR gm.owner=$1))`, uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*repo.Game{}, nil
		}

		return nil, err
	}

	mp := make(map[uint64]*repo.Game, len(plays)/5)
	for _, v := range plays {
		g, ok := mp[v.ID]
		if !ok {
			g = &repo.Game{
				ID:         v.ID,
				FinishedAt: v.FinishedAt,
				Players:    make([]repo.Player, 0, 5),
			}

			mp[v.ID] = g
		}

		var isWinner bool
		if v.Winner != nil {
			isWinner = *v.Winner == v.UID
		}

		g.Players = append(g.Players, repo.Player{
			ID:       v.UID,
			Name:     v.Name,
			IsOwner: v.Owner == v.UID,
			IsWinner: isWinner,
		})
	}

	result := make([]*repo.Game, 0, len(mp))
	for _, v := range mp {
		result = append(result, v)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].FinishedAt.After(result[j].FinishedAt)
	})

	return result, nil
}
