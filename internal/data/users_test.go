package data

import (
	"os"
	"testing"
)

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
		if user, err = user.AddAuthMethod(&PubKey{Username: "test2", Pub: []byte("pub")}); err != nil {
			t.Fatal(err)
		}
		if len(user.AuthMethods) != 2 {
			t.Fatalf("expected 2 auth methods, got %d", len(user.AuthMethods))
		}
		pk := &PubKey{Username: "test2", Pub: []byte("pub")}
		if err = pk.Authenticate(); err != nil {
			t.Fatal("expected pub key to authenticate")
		}
		if user, err = user.AddAuthMethod(&PubKey{Username: "test2", Pub: []byte("pub2")}); err != nil {
			t.Fatal(err)
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
		auth := &PubKey{
			Username: "test2",
			Pub:      []byte("pub"),
		}
		if err = auth.Authenticate(); err != nil {
			t.Fatalf("expected auth to succeed, got: %v", err)
		}
		auth.Pub = []byte("asdjfas")
		if err = auth.Authenticate(); err == nil {
			t.Fatal("expected auth to fail")
		}
	})
	t.Run("DelPubKey", func(t *testing.T) {
		user, err := GetUser("test2")
		if err != nil {
			t.Fatal(err)
		}
		if user, err = user.DelPubKey([]byte("fdsafdas")); err == nil {
			t.Fatal("expected error deleting non-existent key")
		}
		if user == nil {
			t.Fatal("expected user to not be nil")
		}
		if user, err = user.DelPubKey([]byte("pub2")); err != nil {
			t.Fatal(err)
		}
		auth := NewUserPass(false, "test2", "test2")
		if err = auth.Authenticate(); err != nil {
			t.Fatalf("expected userpass to still be there after deleting public key, got: %v", err)
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
