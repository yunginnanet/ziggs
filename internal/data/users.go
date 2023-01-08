package data

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog/log"
)

var ErrAccessDenied = errors.New("access denied")

type User struct {
	Username    string `json:"username"`
	AuthMethods []any  `json:"auth_methods"`
	authMethods []AuthMethod
}

type UserPass struct {
	Username string `json:"pass_username"`
	Password string `json:"password"`
}

func (u *UserPass) Name() string {
	return "password"
}

func (u *UserPass) Authenticate() error {
	user, err := GetUser(u.Username)
	if err != nil {
		log.Warn().Err(err).Str("username", u.Username).Msg("error getting user")
		FakeCycle()
		return ErrAccessDenied
	}
	for _, method := range user.authMethods {
		if method.Name() == "password" {
			userPass := method.(*UserPass)
			if userPass.Password == u.Password {
				return nil
			}
		}
	}
	return ErrAccessDenied
}

type PubKey struct {
	Username string `json:"pubusername"`
	Pub      []byte `json:"pubkey"`
}

func (p PubKey) Name() string {
	return "publickey"
}

func (p PubKey) Authenticate() error {
	u, err := GetUser(p.Username)
	if err != nil {
		log.Warn().Err(err).Str("username", p.Username).Msg("error getting user")
		FakeCycle()
		return ErrAccessDenied
	}
	for _, method := range u.authMethods {
		if method.Name() == "publickey" {
			pubKey := method.(*PubKey)
			if bytes.Equal(pubKey.Pub, p.Pub) {
				return nil
			}
		}
	}
	return ErrAccessDenied
}

type AuthMethod interface {
	// Name returns the name of the authentication method.
	Name() string
	// Authenticate authenticates the user.
	Authenticate() error
}

func GetUser(username string) (*User, error) {
	res, err := db.With("users").Get([]byte(username))
	if err != nil {
		return nil, err
	}
	var user User
	if err := json.Unmarshal(res, &user); err != nil {
		return nil, err
	}
	for _, method := range user.AuthMethods {
		var up UserPass
		var pk PubKey
		jm, err := json.Marshal(method)
		if err != nil {
			return nil, err
		}
		if uperr := json.Unmarshal(jm, &up); uperr == nil {
			if up.Username == "" || up.Password == "" {
				continue
			}
			user.authMethods = append(user.authMethods, &up)
			user.AuthMethods = append(user.AuthMethods, &up)
		}
		if pkerr := json.Unmarshal(jm, &pk); pkerr == nil {
			if pk.Username == "" || len(pk.Pub) == 0 {
				continue
			}
			user.authMethods = append(user.authMethods, &pk)
			user.AuthMethods = append(user.AuthMethods, &pk)
		}
	}
	return &user, nil
}

func NewUser(username string, authMethods ...AuthMethod) error {
	if len(username) == 0 {
		return errors.New("username cannot be empty")
	}
	if len(authMethods) == 0 {
		return errors.New("at least one authentication method must be provided")
	}
	var methods []AuthMethod
	var jsonMethods []any
	for _, method := range authMethods {
		if method == nil {
			return errors.New("authentication method cannot be nil")
		}
		methods = append(methods, method)
		jsonMethods = append(jsonMethods, method)
	}
	user := &User{
		Username:    username,
		AuthMethods: jsonMethods,
	}
	b, err := json.Marshal(user)
	if err != nil {
		return err
	}
	spew.Dump(b)
	return db.With("users").Put([]byte(username), b)
}

func (user *User) AddAuthMethod(method AuthMethod) error {
	if method == nil {
		return errors.New("authentication method cannot be nil")
	}
	user.authMethods = append(user.authMethods, method)
	user.AuthMethods = append(user.AuthMethods, method)
	b, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return db.With("users").Put([]byte(user.Username), b)
}

func DelUser(username string) error {
	return db.With("users").Delete([]byte(username))
}

func (user *User) DelPubKey(pubkey []byte) error {
	var found = false
	var jsonMethods []any
	var methods []AuthMethod
	for _, method := range user.authMethods {
		if method.Name() == "publickey" {
			pubKey := method.(*PubKey)
			if bytes.Equal(pubKey.Pub, pubkey) {
				found = true
				continue
			}
		}
		methods = append(methods, method)
		jsonMethods = append(jsonMethods, method)
	}
	if !found {
		return errors.New("public key not found")
	}
	user.AuthMethods = jsonMethods
	user.authMethods = methods
	if b, err := json.Marshal(user); err == nil {
		return db.With("users").Put([]byte(user.Username), b)
	} else {
		return err
	}
}

func (user *User) ChangePassword(newPassword string) error {
	var ponce = &sync.Once{}
	var methods []any
	var authMethods []AuthMethod
	for _, method := range user.authMethods {
		if method.Name() == "password" {
			ponce.Do(func() {
				method.(*UserPass).Password = newPassword
			})
		}
		methods = append(methods, method)
		authMethods = append(authMethods, method)
	}
	user.AuthMethods = methods
	user.authMethods = authMethods
	b, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return db.With("users").Put([]byte(user.Username), b)
}

func provisionFakeUser() *User {
	err := NewUser("0", &UserPass{Password: "0"})
	if err != nil {
		log.Panic().Err(err).Msg("error creating fake user")
	}
	var user *User
	user, err = GetUser("0")
	if err != nil {
		log.Panic().Err(err).Msg("error getting user")
	}
	return user
}

// FakeCycle chooses the first known user and cycles through all their auth methods to avoid time based user enumeration.
func FakeCycle() {
	user, err := GetUser("0")
	if err != nil {
		user = provisionFakeUser()
	}

	for _, method := range user.authMethods {
		_ = method.Authenticate()
	}
}
