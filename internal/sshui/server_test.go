package sshui

import (
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"git.tcp.direct/kayos/ziggs/internal/config"
	"git.tcp.direct/kayos/ziggs/internal/data"
)

func TestServeSSH(t *testing.T) {
	config.Init()
	data.StartTest()
	go func() {
		t.Log("Starting SSH server")
		err := ServeSSH()
		if err != nil {
			t.Error(err)
		}
	}()
	time.Sleep(2 * time.Second)
	_, err := data.NewUser("test", data.NewUserPass(true, "test", "test"))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("GoodLoginPassword", func(t *testing.T) {
		client, err := ssh.Dial("tcp", config.SSHListen, &ssh.ClientConfig{
			User: "test",
			Auth: []ssh.AuthMethod{
				ssh.Password("test"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		})
		if err != nil {
			t.Fatal(err)
		}
		session, err := client.NewSession()
		if err != nil {
			t.Error(err)
		}
		session.Close()
		client.Close()
	})
	t.Run("BadLoginPassword", func(t *testing.T) {
		client, err := ssh.Dial("tcp", config.SSHListen, &ssh.ClientConfig{
			User: "test",
			Auth: []ssh.AuthMethod{
				ssh.Password("yeet"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if client != nil {
			client.Close()
		}
	})
}
