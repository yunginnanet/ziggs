package sshui

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"

	"github.com/gliderlabs/ssh"

	"git.tcp.direct/kayos/ziggs/internal/config"
	"git.tcp.direct/kayos/ziggs/internal/data"
)

func ServeSSH() error {
	var opts []ssh.Option

	switch config.SSHHostKey {
	case "":
		privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return err
		}
		if err = privateKey.Validate(); err != nil {
			return err
		}
		dir, _ := filepath.Split(config.Filename)
		newFile := filepath.Join(dir, "host_rsa")
		if err = os.WriteFile(newFile, encodePrivateKeyToPEM(privateKey), 0600); err != nil {
			return err
		}
		config.Snek.Set("ssh.host_key", newFile)
	default:
		opts = append(opts, ssh.HostKeyFile(config.SSHHostKey))
	}

	opts = append(opts, ssh.PasswordAuth(func(ctx ssh.Context, password string) bool {
		attempt := data.NewUserPass(false, ctx.User(), password)
		err := attempt.Authenticate()
		return err == nil
	}))

	return ssh.ListenAndServe(config.SSHListen, nil, opts...)
}
