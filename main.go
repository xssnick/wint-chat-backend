package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/xssnick/goeasy/server"

	"github.com/xssnick/wint/pkg/cryptor"
	"github.com/xssnick/wint/pkg/gaming"
	"github.com/xssnick/wint/pkg/matchmaker"
	"github.com/xssnick/wint/pkg/notify"
	"github.com/xssnick/wint/pkg/otp"
	"github.com/xssnick/wint/pkg/photomgr"
	"github.com/xssnick/wint/pkg/poller"
	"github.com/xssnick/wint/pkg/repo/postgres"
	"github.com/xssnick/wint/pkg/typing"
	"github.com/xssnick/wint/service"

	_ "github.com/lib/pq"
)

func main() {
	srv := server.New(server.Config{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
	})

	pg, err := postgres.NewPostgresRepo(postgres.Config{
		Host:     "localhost",
		Login:    "test",
		Password: "mysecret",
		DB:       "wint",
		Port:     54320,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer pg.Close()

	cryp, err := cryptor.NewAESGCM(make([]byte, 32))
	if err != nil {
		log.Fatalln(err)
	}

	totp := otp.NewTOTP(sha256.New, []byte("secret228"))

	imgr := photomgr.New("./images")

	matcher := matchmaker.NewMatchmaker(matchmaker.Config{
		Repo:          pg,
		PingValid:     7,
		MaxPlayers:    7,
		MinPlayers:    1,
		LongSearchSec: 120,
		TickEvery:     300 * time.Millisecond,
	})

	poll := poller.NewPoller()
	poll.StartCleaner(1 * time.Second)

	tpr := typing.NewTyper(poll)

	gmg := gaming.NewGaming(1*time.Second, pg, poll)
	gmg.StartCheckGames()

	go tpr.Cleaner()
	go matcher.LinkRoutine()

	svc := &service.Service{
		Poller:     poll,
		Gaming:     gmg,
		Typer:      tpr,
		Repo:       pg,
		OTP:        totp,
		Cryptor:    cryp,
		Notifier:   notify.NewSMSNotifier(),
		Matchmaker: matcher,
		ImgManager: imgr,
	}

	srv.MustRegister(fasthttp.MethodGet, "/version", &service.Info{Service: svc, Version: "1.1"})
	srv.MustRegister(fasthttp.MethodPost, "/profile/edit", &service.ProfileEdit{Service: svc})
	srv.MustRegister(fasthttp.MethodGet, "/profile", &service.Profile{Service: svc})
	srv.MustRegister(fasthttp.MethodPost, "/login", &service.Login{Service: svc})
	srv.MustRegister(fasthttp.MethodPost, "/login/finish", &service.LoginEnd{Service: svc})

	srv.MustRegister(fasthttp.MethodPost, "/game/find", &service.FindChat{Service: svc})
	srv.MustRegister(fasthttp.MethodPost, "/game/create", &service.CreateChat{Service: svc})
	srv.MustRegister(fasthttp.MethodGet, "/game/state", &service.StateChat{Service: svc})
	srv.MustRegister(fasthttp.MethodPost, "/game/exit", &service.ExitChat{Service: svc})
	srv.MustRegister(fasthttp.MethodPost, "/game/start", &service.StartChat{Service: svc})
	srv.MustRegister(fasthttp.MethodPost, "/game/kick", &service.KickChat{Service: svc})
	srv.MustRegister(fasthttp.MethodPost, "/game/like", &service.MessageLike{Service: svc})

	srv.MustRegister(fasthttp.MethodPost, "/message/send", &service.MessageSend{Service: svc})
	srv.MustRegister(fasthttp.MethodGet, "/message/get", &service.MessagesGet{Service: svc})
	srv.MustRegister(fasthttp.MethodPost, "/message/type", &service.MessageType{Service: svc})
	srv.MustRegister(fasthttp.MethodPost, "/message/type/stop", &service.MessageTypeStop{Service: svc})

	srv.MustRegister(fasthttp.MethodGet, "/chats/get", &service.ChatsGet{Service: svc})
	srv.MustRegister(fasthttp.MethodGet, "/games/get", &service.GamesGet{Service: svc})

	srv.MustRegister(fasthttp.MethodGet, "/images/:owner/:img", &service.Pictures{Service: svc})

	log.Println(srv.Listen(":7777"))
}

func Hello(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "Hello, %s!\n", ctx.UserValue("name"))
}
