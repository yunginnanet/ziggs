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
		if err := NewUser("test"); err == nil {
			t.Fatal("expected error creating user with no auth method")
		}
		if _, err := GetUser("test"); err == nil {
			t.Fatal("expected error getting user with no auth method")
		}
		if err := NewUser("test", NewUserPass("test", "test")); err != nil {
			t.Fatal(err)
		}
		tu, err := GetUser("test")
		if err != nil {
			t.Fatal(err)
		}
		if len(tu.authMethods) != 1 {
			t.Fatalf("expected 1 auth method, got %d", len(tu.authMethods))
		}
		if tu.authMethods[0].Name() != "password" {
			t.Fatalf("expected auth method to be 'password', got '%s'", tu.authMethods[0].Name())
		}
		if tu.Username != "test" {
			t.Fatalf("expected username to be 'test', got '%s'", tu.Username)
		}
	})
	t.Run("AddAuthMethod", func(t *testing.T) {
		user, err := GetUser("test")
		if err != nil {
			t.Fatal(err)
		}
		if err = user.AddAuthMethod(nil); err == nil {
			t.Fatal("expected error adding nil auth method")
		}
		if err = user.AddAuthMethod(&PubKey{Username: "test", Pub: []byte("test")}); err != nil {
			t.Fatal(err)
		}
		if len(user.authMethods) != 2 {
			t.Fatalf("expected 2 auth methods, got %d", len(user.authMethods))
		}
		if user.authMethods[0].Name() != "password" {
			t.Fatalf("expected auth method to be 'password', got '%s'", user.authMethods[0].Name())
		}
		if user.authMethods[1].Name() != "publickey" {
			t.Fatalf("expected auth method to be 'publickey', got '%s'", user.authMethods[1].Name())
		}
		auth := &PubKey{
			Username: "test",
			Pub:      []byte("test"),
		}
		if err = auth.Authenticate(); err != nil {
			t.Fatalf("expected auth to succeed, got: %v", err)
		}
		auth.Pub = []byte("test2")
		if err = auth.Authenticate(); err == nil {
			t.Fatal("expected auth to fail")
		}
	})
	t.Run("DelPubKey", func(t *testing.T) {
		user, err := GetUser("test")
		if err != nil {
			t.Fatal(err)
		}
		if err = user.DelPubKey([]byte("test2")); err == nil {
			t.Fatal("expected error deleting non-existent key")
		}
		if err = user.DelPubKey([]byte("test")); err != nil {
			t.Fatal(err)
		}
		auth := NewUserPass("test", "test")
		if err := auth.Authenticate(); err != nil {
			t.Fatalf("expected userpass to still be there after deleting public key, got: %v", err)
		}
	})
	t.Run("ChangePassword", func(t *testing.T) {
		user, err := GetUser("test")
		if err != nil {
			t.Fatal(err)
		}
		if err = user.ChangePassword("test2"); err != nil {
			t.Fatal(err)
		}
		auth := NewUserPass("test", "test")
		if err = auth.Authenticate(); err == nil {
			t.Fatal("expected auth to fail using old password")
		}
		auth = NewUserPass("test", "test2")
		if err = auth.Authenticate(); err != nil {
			t.Fatalf("expected auth to succeed using new password, got: %v", err)
		}
	})
}
