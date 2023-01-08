package sshui

import (
	"log"

	"github.com/gliderlabs/ssh"

	"git.tcp.direct/kayos/ziggs/internal/config"
)

func ServeSSH() {
	var opts []ssh.Option

	if config.SSHHostKey != "" {
		opts = append(opts, ssh.HostKeyFile(config.SSHHostKey))
	}

	opts = append(opts, ssh.PasswordAuth(func(ctx ssh.Context, password string) bool {

		return false
	}))

	log.Fatal(ssh.ListenAndServe(":2222", nil, opts...))
}
