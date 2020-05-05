package service

import (
	"context"

	"github.com/xssnick/goeasy"

	"github.com/xssnick/wint/pkg/cryptor"
	"github.com/xssnick/wint/pkg/repo"
)

type Gaming interface {
	CanWriteLeft(uid uint64, mstate *repo.MatchState, msgs []repo.Message) (bool, bool, bool, int64)
}

type Matchmaker interface {
	StartMatch(user uint64) error
	FindMatch(user uint64, mode int) error
	CreateMatch(user uint64, mode int) error
	StateMatch(user uint64, withPing bool) (repo.MatchState, error)
	ExitMatch(user uint64) error
}

type ImageManager interface {
	SaveImage(data []byte, user uint64, name string) error
	GetImage(user uint64, name string) ([]byte, error)
	ListImages(user uint64) ([]string, error)
}

type Notifier interface {
	Notify(phone uint64, code string) error
}

type OTP interface {
	Generate(base string, digs int) (long []byte, short string)
	Validate(base string, digs int, slong []byte, sshort string) bool
}

type Typer interface {
	SetType(on, user uint64)
	GetTypes(on uint64, users []uint64) []uint64
	DelType(on, user uint64)
}

type Poller interface {
	ForgotEvent(id uint64, ch chan bool)
	WaitEvent(e uint64, id uint64) (chan bool, bool)
	PushEvent(id uint64)
	GetEvent(id uint64) string
}

type Repo interface {
	GetUserIDByPhone(ctx context.Context, phone uint64) (uint64, error)
	CreateUser(ctx context.Context, phone uint64) (uint64, error)
	EditProfile(ctx context.Context, id uint64, data repo.UserEdit) error
	GetUserProfile(ctx context.Context, id uint64) (*repo.UserProfile, error)
	GetMessagesGame(ctx context.Context, gameid uint64) ([]repo.Message, error)
	SendMessage(ctx context.Context, owner uint64, isgame bool, target uint64, text string) (uint64, error)
	GetGameStateByID(ctx context.Context, id, uid uint64, unsafe bool) (repo.MatchState, error)
	GetUsersNames(ctx context.Context, uids []uint64) ([]repo.UserNameID, error)
	KickFromGame(ctx context.Context, owner, game, user uint64) error
	FinishGame(ctx context.Context, owner, game, winner uint64) error
	GetLikesGame(ctx context.Context, gameid uint64) ([]repo.Like, error)
	GetGames(ctx context.Context, uid uint64) ([]*repo.Game, error)

	SetLike(ctx context.Context, owner uint64, msgid uint64) error
	DeleteLike(ctx context.Context, owner uint64, msgid uint64) error

	IsGameMember(ctx context.Context, id, uid uint64) (bool, error)

	GetMessagesChat(ctx context.Context, me, you uint64) ([]repo.Message, error)
	GetChats(ctx context.Context, me uint64, search string) ([]repo.Chat, error)
	SetRead(ctx context.Context, me, you uint64) error
}

type Service struct {
	goeasy.BasicService

	Repo       Repo
	OTP        OTP
	Notifier   Notifier
	Cryptor    *cryptor.AESGCM
	ImgManager ImageManager
	Matchmaker Matchmaker
	Typer      Typer
	Gaming     Gaming
	Poller     Poller
}

func (b *Service) MiddlewareChain() []goeasy.HttpMiddleware {
	return []goeasy.HttpMiddleware{&Auth{Service: b}}
}
