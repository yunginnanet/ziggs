package data

import (
	"encoding/json"
	"errors"
	"sync"

	"git.tcp.direct/kayos/common/entropy"
	"git.tcp.direct/kayos/common/squish"
	"github.com/davecgh/go-spew/spew"
	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"
)

type StringMapper interface {
	Map() map[string]string
}

type AuthMethod interface {
	json.Marshaler
	StringMapper
	// Authenticate authenticates the user.
	Authenticate() error
	// Name returns the name of the authentication method.
	Name() string
}

func AuthMethodFromMap(m map[string]string) AuthMethod {
	switch m["type"] {
	case "password":
		return &UserPass{
			Username: m["pass_username"],
			Password: m["password"],
		}
	case "publickey":
		pkParsed, err := ssh.ParsePublicKey(squish.B64d(m["pubkey"]))
		if err != nil {
			return nil
		}
		return &PubKey{
			Username: m["pub_username"],
			Pub:      pkParsed,
		}
	}
	return nil
}

var ErrAccessDenied = errors.New("access denied")

type User struct {
	Username    string              `json:"username"`
	AuthMethods []map[string]string `json:"auth_methods"`
	*sync.Mutex
}

type UserPass struct {
	Username string `json:"pass_username"`
	Password string `json:"password"`
}

func (up *UserPass) Name() string {
	return "password"
}

func (up *UserPass) Map() map[string]string {
	return map[string]string{
		"type":          "password",
		"pass_username": up.Username,
		"password":      up.Password,
	}
}

func (up *UserPass) MarshalJSON() ([]byte, error) {
	return json.Marshal(up.Map())
}

func NewUserPass(hashIt bool, username, password string) *UserPass {
	var input = password
	var err error
	if hashIt {
		input, err = HashPassword(password)
		if err != nil {
			panic(err)
		}
	}
	return &UserPass{
		Username: username,
		Password: input,
	}
}

func (up *UserPass) Authenticate() error {
	if up.Username == "" {
		return errors.New("username cannot be empty")
	}
	user, err := GetUser(up.Username)
	if err != nil {
		log.Warn().Err(err).Str("username", up.Username).Msg("error getting user")
		FakeCycle()
		return ErrAccessDenied
	}
	for _, method := range user.AuthMethods {
		switch method["type"] {
		case "password":
			if method["pass_username"] == up.Username && CheckPasswordHash(up.Password, method["password"]) {
				return nil
			}
		default:
			continue
		}
	}
	return ErrAccessDenied
}

type PubKey struct {
	Username string        `json:"pub_username"`
	Pub      ssh.PublicKey `json:"pubkey"`
}

func NewPubKey(username string, pubkey ssh.PublicKey) *PubKey {
	return &PubKey{
		Username: username,
		Pub:      pubkey,
	}
}

func (pk *PubKey) Name() string {
	return "publickey"
}

func (pk *PubKey) Map() map[string]string {
	return map[string]string{
		"type":         "publickey",
		"pub_username": pk.Username,
		"pubkey":       squish.B64e(pk.Pub.Marshal()),
	}
}

func (pk *PubKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(pk.Map())
}

func (pk *PubKey) Authenticate() error {
	user, err := GetUser(pk.Username)
	if err != nil {
		log.Warn().Err(err).Str("username", pk.Username).Msg("error getting user")
		FakeCycle()
		return ErrAccessDenied
	}
	for _, method := range user.AuthMethods {
		switch method["type"] {
		case "publickey":
			if method["pub_username"] != pk.Username {
				log.Warn().Str("username", pk.Username).Msg("username mismatch")
				continue
			}
			pkdat, ok := method["pubkey"]
			if !ok {
				log.Warn().Str("username", pk.Username).Msg("pubkey not found")
				continue
			}
			pubkeyParsed, err := ssh.ParsePublicKey(squish.B64d(pkdat))
			if err != nil {
				log.Warn().Err(err).Str("username", pk.Username).Msg("error parsing public key")
				spew.Dump(method)
				continue
			}
			if ssh.KeysEqual(pubkeyParsed, pk.Pub) {
				return nil
			}
		default:
			continue
		}
	}
	return ErrAccessDenied
}

func GetUser(username string) (*User, error) {
	res, err := db.With("users").Get([]byte(username))
	if err != nil {
		return nil, err
	}
	var user User
	if err = json.Unmarshal(res, &user); err != nil {
		return nil, err
	}
	user.Mutex = &sync.Mutex{}
	return &user, nil
}

func NewUser(username string, authMethods ...AuthMethod) (*User, error) {
	if len(username) == 0 {
		return nil, errors.New("username cannot be empty")
	}
	if len(authMethods) == 0 {
		return nil, errors.New("at least one authentication method must be provided")
	}
	var methods []map[string]string
	for _, method := range authMethods {
		if method == nil {
			return nil, errors.New("authentication method cannot be nil")
		}
		switch method.Name() {
		case "password":
			usableMethod := method.(*UserPass)
			if len(usableMethod.Username) == 0 {
				return nil, errors.New("username cannot be empty")
			}
			if len(usableMethod.Password) == 0 {
				return nil, errors.New("password cannot be empty")
			}
			methods = append(methods, method.Map())
		case "publickey":
			usableMethod := method.(*PubKey)
			if len(usableMethod.Username) == 0 {
				return nil, errors.New("username cannot be empty")
			}
			if len(usableMethod.Pub.Marshal()) == 0 {
				return nil, errors.New("public key cannot be empty")
			}
			methods = append(methods, method.Map())
		}
	}
	if len(methods) == 0 {
		return nil, errors.New("at least one authentication method must be provided")
	}
	user := &User{
		Username:    username,
		AuthMethods: methods,
		Mutex:       &sync.Mutex{},
	}
	b, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	return user, db.With("users").Put([]byte(username), b)
}

func (user *User) AddAuthMethod(method AuthMethod) (*User, error) {
	user.Lock()
	defer user.Unlock()
	if method == nil {
		return user, errors.New("authentication method cannot be nil")
	}
	user.AuthMethods = append(user.AuthMethods, method.Map())
	b, err := json.Marshal(user)
	if err != nil {
		return user, err
	}
	return user, db.With("users").Put([]byte(user.Username), b)
}

func DelUser(username string) error {
	return db.With("users").Delete([]byte(username))
}

func (user *User) DelPubKey(pubkey ssh.PublicKey) (*User, error) {
	user.Lock()
	defer user.Unlock()
	var found = false
	var methods []map[string]string
	for _, method := range user.AuthMethods {
		m := AuthMethodFromMap(method)
		if m == nil {
			continue
		}
		if m.Name() == "publickey" {
			pubKey := m.(*PubKey)
			if ssh.KeysEqual(pubKey.Pub, pubkey) {
				found = true
				continue
			}
		}
		methods = append(methods, method)
	}
	if !found {
		return user, errors.New("public key not found")
	}
	user.AuthMethods = methods
	if b, err := json.Marshal(user); err == nil {
		return user, db.With("users").Put([]byte(user.Username), b)
	} else {
		return user, err
	}
}

func (user *User) ChangePassword(newPassword string) (*User, error) {
	user.Lock()
	defer user.Unlock()
	var ponce = &sync.Once{}
	var methods []map[string]string
	for _, method := range user.AuthMethods {
		m := AuthMethodFromMap(method)
		if m == nil {
			continue
		}
		if m.Name() == "password" {
			ponce.Do(func() {
				hashed, err := HashPassword(newPassword)
				if err != nil {
					panic(err)
				}
				m.(*UserPass).Password = hashed
			})
		}
		methods = append(methods, m.Map())
	}
	user.AuthMethods = methods
	b, err := json.Marshal(user)
	if err != nil {
		return user, err
	}
	return user, db.With("users").Put([]byte(user.Username), b)
}

func provisionFakeUser() *User {
	user, err := NewUser("0", NewUserPass(true, "0", entropy.RandStrWithUpper(32)))
	if err != nil {
		log.Panic().Err(err).Msg("error creating fake user")
	}
	return user
}

// FakeCycle chooses the first known user and cycles through all their auth methods to avoid time based user enumeration.
func FakeCycle() {
	user, err := GetUser("0")
	if err != nil {
		user = provisionFakeUser()
	}

	for n, method := range user.AuthMethods {
		if n > 2 {
			break
		}
		_ = AuthMethodFromMap(method).Authenticate()
	}
}
