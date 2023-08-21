package ssh

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.tcp.direct/kayos/common/squish"
	"github.com/rs/zerolog"

	"git.tcp.direct/kayos/ziggs/internal/config"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
)

type Server struct {
	*ssh.Server
	clients map[string]ssh.Session
	keys    map[string]ssh.PublicKey
	log     *zerolog.Logger
}

func (s *Server) checkAuth(h ssh.Handler) ssh.Handler {
	return func(ss ssh.Session) {
		k, ok := s.GetStaticKeys()[squish.B64e(ss.PublicKey().Marshal())]
		if !ok {
			wish.Println(ss, "huh?")
			_ = ss.Close()
			return
		}
		switch {
		case ssh.KeysEqual(ss.PublicKey(), k):
			h(ss)
		default:
			wish.Println(ss, "FUBAR")
			_ = ss.Close()
			return
		}
	}
}

func (s *Server) ziggssh(h ssh.Handler) ssh.Handler {
	return func(ss ssh.Session) {
		ss.
			cli.StartCLI()
		h(ss)
	}
}

func (s *Server) ListenAndServe() error {
	var err error
	s.Server, err = wish.NewServer(
		wish.WithAddress(config.SSHListen),
		wish.WithHostKeyPath(config.SSHHostKey),
		wish.WithVersion("SSH-2.0-ziggs"),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		}),
		wish.WithMiddleware(
			logging.Middleware(),
			s.checkAuth,
		),
	)
	if err != nil {
		return err
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Starting SSH server on %s", config.SSHListen)
	go func() {
		if err = s.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	go func() {
		<-done
		log.Println("Stopping SSH server")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer func() { cancel() }()
		if err := s.Shutdown(ctx); err != nil {
			log.Fatalln(err)
		}
	}()

	return nil
}
