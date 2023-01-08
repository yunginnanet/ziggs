package sshui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gliderlabs/ssh"

	"git.tcp.direct/kayos/ziggs/internal/config"
	"git.tcp.direct/kayos/ziggs/internal/data"
)

func newHostKey() error {
	privateKey, err := generatePrivateKey()
	if err != nil {
		return err
	}
	dir, _ := filepath.Split(config.Filename)
	newFile := filepath.Join(dir, "host_rsa")
	if err = os.WriteFile(newFile, encodePrivateKeyToPEM(privateKey), 0600); err != nil {
		return err
	}
	config.Snek.Set("ssh.host_key", newFile)
	config.SSHHostKey = newFile
	if err = config.Snek.WriteConfig(); err != nil {
		return fmt.Errorf("viper config save error: %v", err)
	}
	return nil
}

func ServeSSH() error {
	var opts []ssh.Option

	if config.SSHHostKey == "" {
		if err := newHostKey(); err != nil {
			return err
		}
	}

	opts = append(opts, ssh.HostKeyFile(config.SSHHostKey))

	opts = append(opts, ssh.PasswordAuth(func(ctx ssh.Context, password string) bool {
		attempt := data.NewUserPass(false, ctx.User(), password)
		err := attempt.Authenticate()
		return err == nil
	}))

	opts = append(opts, ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		attempt := data.NewPubKey(ctx.User(), key)
		err := attempt.Authenticate()
		return err == nil
	}))

	return ssh.ListenAndServe(config.SSHListen, nil, opts...)
}
