package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/lib/pq"

	"github.com/xssnick/wint/pkg/repo"
)

func (r *Repo) CreateGame(ctx context.Context, owner uint64, users []uint64) (uint64, error) {
	if len(users) == 0 {
		return 0, errors.New("0 users")
	}

	log.Println("making game for user:", owner)
	var id uint64

	err := r.db.GetContext(ctx, &id, "INSERT INTO games (owner,winner,created_at) VALUES ($1,$2,$3) RETURNING id", owner, nil, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	var vals string
	for _, u := range users {
		vals += "(" + strconv.FormatUint(id, 10) + "," + strconv.FormatUint(u, 10) + "),"
	}

	_, err = r.db.ExecContext(ctx, "INSERT INTO plays (game, player) VALUES "+vals[:len(vals)-1])
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repo) GetGameState(ctx context.Context, user uint64) (repo.MatchState, error) {
	var state struct {
		ID        uint64        `db:"id"`
		Owner     uint64        `db:"owner"`
		StartedAt time.Time     `db:"started_at"`
		Players   pq.Int64Array `db:"players"`
	}

	// as owner
	err := r.db.GetContext(ctx, &state, `SELECT g.id as id,g.created_at as started_at, g.owner as owner, ARRAY(
	SELECT player
	FROM plays
	WHERE game = g.id AND lost_at IS NULL) as players FROM games g WHERE owner=$1 AND g.finished_at is NULL`, user)
	if err != nil && err != sql.ErrNoRows {
		return repo.MatchState{}, err
	}

	if err == sql.ErrNoRows {
		err = r.db.GetContext(ctx, &state, `SELECT g.id as id,g.created_at as started_at, g.owner as owner, ARRAY(
	SELECT player
	FROM plays
	WHERE game = g.id AND lost_at IS NULL) as players FROM games g JOIN plays p ON p.game=g.id AND p.player=$1 AND p.lost_at IS NULL WHERE g.finished_at is NULL`, user)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Println("ggs nf", user)
				return repo.MatchState{}, repo.ErrNotFound
			}

			return repo.MatchState{}, err
		}
	}

	var players = make([]uint64, len(state.Players))
	for i, v := range state.Players {
		players[i] = uint64(v)
	}

	return repo.MatchState{
		ID:        state.ID,
		Owner:     state.Owner,
		Players:   players,
		StartedAt: state.StartedAt.Unix(),
	}, nil
}

func (r *Repo) GetGameStateByID(ctx context.Context, id, uid uint64, unsafe bool) (repo.MatchState, error) {
	var state struct {
		ID         uint64        `db:"id"`
		Owner      uint64        `db:"owner"`
		StartedAt  time.Time     `db:"started_at"`
		FinishedAt *time.Time    `db:"finished_at"`
		Players    pq.Int64Array `db:"players"`
	}

	// as owner
	err := r.db.GetContext(ctx, &state, `SELECT g.id as id,g.created_at as started_at, g.owner as owner, g.finished_at as finished_at, ARRAY(
	SELECT player
	FROM plays
	WHERE game = g.id AND lost_at IS NULL) as players FROM games g JOIN plays p ON p.game=$1 WHERE g.id=$1 AND (g.owner=$2 OR p.player=$2 OR $3=true)`, id, uid, unsafe)
	if err != nil {
		if err == sql.ErrNoRows {
			return repo.MatchState{}, repo.ErrNotFound
		}

		return repo.MatchState{}, err
	}

	var players = make([]uint64, len(state.Players))
	for i, v := range state.Players {
		players[i] = uint64(v)
	}

	var fat int64
	if state.FinishedAt != nil {
		fat = state.FinishedAt.Unix()
	}

	return repo.MatchState{
		ID:         state.ID,
		Owner:      state.Owner,
		Players:    players,
		StartedAt:  state.StartedAt.Unix(),
		FinishedAt: fat,
	}, nil
}

func (r *Repo) IsGameMember(ctx context.Context, id, uid uint64) (bool, error) {
	var has int

	err := r.db.GetContext(ctx, &has, `SELECT COUNT(1) FROM games g JOIN plays p ON p.game=g.id WHERE g.id=$1 AND (g.owner=$2 OR p.player=$2)`, id, uid)
	if err != nil {
		return false, err
	}

	return has > 0, nil
}

func (r *Repo) SendMessage(ctx context.Context, owner uint64, isgame bool, target uint64, text string) (uint64, error) {
	var id uint64

	var game, toUser *uint64
	if isgame {
		game = &target
	} else {
		toUser = &target
	}

	err := r.db.GetContext(ctx, &id, "INSERT INTO messages (owner,msg,game,target,created_at) VALUES ($1,$2,$3,$4,$5) RETURNING id", owner, text, game, toUser, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repo) SetLike(ctx context.Context, owner uint64, msgid uint64) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO likes (owner,message) VALUES ($1,$2) ON CONFLICT DO NOTHING", owner, msgid)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) DeleteLike(ctx context.Context, owner uint64, msgid uint64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM likes WHERE owner=$1 AND message=$2", owner, msgid)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) GetMessagesChat(ctx context.Context, me, you uint64) ([]repo.Message, error) {
	var msgs []struct {
		ID        uint64    `db:"id"`
		Name      string    `json:"name"`
		Owner     uint64    `db:"owner"`
		Text      string    `db:"msg"`
		CreatedAt time.Time `db:"created_at"`
	}

	err := r.db.SelectContext(ctx, &msgs, `SELECT m.id as id, m.created_at as created_at, m.owner as owner, m.msg as msg, u.name as name FROM messages m JOIN users u ON u.id=m.owner WHERE (m.target = $1 AND m.owner=$2) OR (m.target=$2 AND m.owner=$1) ORDER BY m.created_at ASC`, me, you)
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


func (r *Repo) GetChats(ctx context.Context, me uint64, search string) ([]repo.Chat, error) {
	var chats []struct {
		ChatID        uint64    `db:"chat"`
		OwnerID        uint64    `db:"owner"`
		Unread        int    `db:"unread"`
		GameID        *uint64    `db:"game"`
		Message      string    `db:"msg"`
		CreatedAt time.Time `db:"created_at"`
	}

	err := r.db.SelectContext(ctx, &chats, `select DISTINCT ON (chat) (case when m.owner=$1 then m.target else m.owner end) as chat, m.msg as msg, m.created_at as created_at, m.owner as owner, g.id as game, (SELECT COUNT(1) FROM messages WHERE target=$1 AND owner=(case when m.owner=$1 then m.target else m.owner end) AND seen=false) as unread from messages m left join games g on (g.winner=m.owner) where (m.owner=$1 AND m.game is null) OR m.target=$1 ORDER BY chat, m.created_at DESC`, me)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}

		return nil, err
	}

	result := make([]repo.Chat, len(chats))
	for i, v := range chats {
		result[i] = repo.Chat{
			ID:        v.ChatID,
			Message:   v.Message,
			MessageOwner:     v.OwnerID,
			MessageTime:      v.CreatedAt,
			IsSame: v.GameID != nil,
			Unread: v.Unread,
		}
	}

	return result, nil
}

func (r *Repo) SetRead(ctx context.Context, me, you uint64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE messages SET seen=true WHERE owner=$2 and target=$1 AND seen=false`, me, you)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) GetUnread(ctx context.Context, me uint64) (int,error) {
	var num int

	err := r.db.GetContext(ctx,&num, `SELECT COUNT(1) FROM messages WHERE target=$1 AND seen=false`, me)
	if err != nil {
		return 0,err
	}

	return num, nil
}