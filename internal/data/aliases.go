package data

import (
	"strings"

	"git.tcp.direct/tcp.direct/database/bitcask"
)

func kva() bitcask.Store {
	return kv().With("aliases")
}

func AddAlias(alias, command string) error {
	return kva().Put([]byte(strings.ToLower(strings.TrimSpace(alias))), []byte(command))
}

func GetAlias(alias, command string) (cmd string) {
	a, err := kva().Get([]byte(strings.ToLower(strings.TrimSpace(alias))))
	if err == nil {
		cmd = string(a)
	}
	return
}
