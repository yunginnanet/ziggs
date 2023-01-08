package data

import (
	"os"
	"testing"

	"golang.org/x/crypto/ssh"
)

var (
	testPublicKey1 ssh.PublicKey
	testPublicKey2 ssh.PublicKey
	testPublicKey3 ssh.PublicKey
)

func init() {
	var err error
	// generate public keys for testing
	testPublicKey1, _, _, _, err = ssh.ParseAuthorizedKey([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIO6EFqmelEJ6MELBPHUEFTGmlJBfhS7Jeq5B5BCrFSun"))
	if err != nil {
		panic(err)
	}
	testPublicKey2, _, _, _, err = ssh.ParseAuthorizedKey([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIH+ZTIMTWwYWHUEJlHfhT7dcYhgETGWgwEpDLdURaTPb"))
	if err != nil {
		panic(err)
	}
	testPublicKey3, _, _, _, err = ssh.ParseAuthorizedKey([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHUEFpqqYCfBkVLRwgYlGbZyzgnEcMLpT0o97JUHNpIt"))
	if err != nil {
		panic(err)
	}
}

func TestUsers(t *testing.T) {
	testMode()
	Start()
	t.Cleanup(func() {
		if err := os.RemoveAll(testLocation); err != nil {
			panic(err)
		}
	})
	t.Run("NewUser", func(t *testing.T) {
		if _, err := NewUser("test1"); err == nil {
			t.Fatal("expected error creating user with no auth method")
		}
		if _, err := GetUser("test1"); err == nil {
			t.Fatal("expected error getting user with no auth method")
		}
		if _, err := NewUser("test1", NewUserPass(true, "test", "test")); err != nil {
			t.Fatal(err)
		}
		tu, err := GetUser("test1")
		if err != nil {
			t.Fatal(err)
		}
		if len(tu.AuthMethods) != 1 {
			t.Fatalf("expected 1 auth method, got %d", len(tu.AuthMethods))
		}
		if tu.AuthMethods[0]["type"] != "password" {
			t.Fatalf("expected auth method to be 'password', got '%s'", tu.AuthMethods[0]["type"])
		}
		if tu.Username != "test1" {
			t.Fatalf("expected username to be 'test', got '%s'", tu.Username)
		}
	})
	t.Run("AddAuthMethod", func(t *testing.T) {
		user, err := NewUser("test2", NewUserPass(true, "test2", "test2"))
		if err != nil {
			t.Fatal(err)
		}
		if user, err = user.AddAuthMethod(nil); err == nil {
			t.Fatal("expected error adding nil auth method")
		}
		if user == nil {
			t.Fatal("expected user to not be nil")
		}
		if user, err = user.AddAuthMethod(NewPubKey(user.Username, testPublicKey1)); err != nil {
			t.Fatal(err)
		}
		if len(user.AuthMethods) != 2 {
			t.Fatalf("expected 2 auth methods, got %d", len(user.AuthMethods))
		}
		pk := NewPubKey("test2", testPublicKey1)
		if err = pk.Authenticate(); err != nil {
			t.Fatal("expected pub key 1 to authenticate")
		}
		pk = NewPubKey("test2", testPublicKey2)
		if err = pk.Authenticate(); err == nil {
			t.Fatal("expected pub key 2 to not authenticate")
		}
		if user, err = user.AddAuthMethod(NewPubKey(user.Username, testPublicKey2)); err != nil {
			t.Fatal(err)
		}

		pk = NewPubKey("test2", testPublicKey1)
		if err = pk.Authenticate(); err != nil {
			t.Fatal("expected pub key 1 to authenticate")
		}

		pk = NewPubKey("test2", testPublicKey2)
		if err = pk.Authenticate(); err != nil {
			t.Fatal("expected pub key 2 to authenticate")
		}
		if len(user.AuthMethods) != 3 {
			t.Fatalf("expected 2 auth methods, got %d", len(user.AuthMethods))
		}
		if user.AuthMethods[0]["type"] != "password" {
			t.Fatalf("expected auth method to be 'password', got '%s'", user.AuthMethods[0]["type"])
		}
		if user.AuthMethods[1]["type"] != "publickey" {
			t.Fatalf("expected auth method to be 'publickey', got '%s'", user.AuthMethods[1]["type"])
		}
	})
	t.Run("DelPubKey", func(t *testing.T) {
		user, err := GetUser("test2")
		if err != nil {
			t.Fatal(err)
		}
		if user, err = user.DelPubKey(testPublicKey3); err == nil {
			t.Fatal("expected error deleting non-existent key")
		}
		if user == nil {
			t.Fatal("expected user to not be nil")
		}
		if user, err = user.DelPubKey(testPublicKey2); err != nil {
			t.Fatal(err)
		}
		auth := NewUserPass(false, "test2", "test2")
		if err = auth.Authenticate(); err != nil {
			t.Fatalf("expected userpass to still be there after deleting public key, got: %v", err)
		}
		pk := &PubKey{"test2", testPublicKey2}
		if err = pk.Authenticate(); err == nil {
			t.Fatal("expected public key 2 to be deleted")
		}
		pk = &PubKey{"test2", testPublicKey1}
		if err = pk.Authenticate(); err != nil {
			t.Fatal("expected public key 1 to not be deleted")
		}
	})
	t.Run("ChangePassword", func(t *testing.T) {
		user, err := GetUser("test2")
		if err != nil {
			t.Fatal(err)
		}
		if user, err = user.ChangePassword("test5"); err != nil {
			t.Fatal(err)
		}
		auth := NewUserPass(false, "test2", "test2")
		if err = auth.Authenticate(); err == nil {
			t.Fatal("expected auth to fail using old password")
		}
		auth = NewUserPass(false, "test2", "test5")
		if err = auth.Authenticate(); err != nil {
			t.Fatalf("expected auth to succeed using new password, got: %v", err)
		}
	})
}
