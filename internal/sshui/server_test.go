package sshui

import (
	"crypto/rsa"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"git.tcp.direct/kayos/ziggs/internal/config"
	"git.tcp.direct/kayos/ziggs/internal/data"
)

var (
	testKey1 *rsa.PrivateKey
	testKey2 *rsa.PrivateKey
)

func init() {
	var err error
	// generate public keys for testing
	if testKey1, err = generatePrivateKey(); err != nil {
		panic(err)
	}
	if testKey2, err = generatePrivateKey(); err != nil {
		panic(err)
	}
}

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
	time.Sleep(1250 * time.Millisecond)
	user, err := data.NewUser("test", data.NewUserPass(true, "test", "test"))
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
	t.Run("GoodLoginKey", func(t *testing.T) {
		var signer ssh.Signer
		if signer, err = ssh.NewSignerFromKey(testKey1); err != nil {
			t.Fatal(err)
		}
		if _, err = user.AddAuthMethod(data.NewPubKey(user.Username, signer.PublicKey())); err != nil {
			t.Fatal(err)
		}
		var client *ssh.Client
		if client, err = ssh.Dial("tcp", config.SSHListen, &ssh.ClientConfig{
			User: "test",
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}); err != nil {
			t.Fatal("expected nil when authing with known key, got", err)
		}
		var session *ssh.Session
		if session, err = client.NewSession(); err != nil {
			t.Error(err)
		}
		session.Close()
		client.Close()
	})
	t.Run("BadLoginKey", func(t *testing.T) {
		var signer ssh.Signer
		if signer, err = ssh.NewSignerFromKey(testKey2); err != nil {
			t.Fatal(err)
		}
		var client *ssh.Client
		if client, err = ssh.Dial("tcp", config.SSHListen, &ssh.ClientConfig{
			User: "test",
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}); err == nil {
			t.Fatal("expected error when authing with unknown key, got nil")
		}
		if client != nil {
			client.Close()
		}
	})
}
