package data

import (
	"os"
	"strconv"
	"sync"
	"time"

	"git.tcp.direct/tcp.direct/database/bitcask"
	"github.com/rs/zerolog/log"

	"git.tcp.direct/kayos/ziggs/internal/common"
)

var (
	stores = []string{"macros"}
	istest = false
	once   = &sync.Once{}
	target string
	db     *bitcask.DB
)

func testMode() {
	istest = true
}

func setTarget() {
	if !istest {
		target = common.Home + "/.local/share/" + common.Title + "/"
	}
	target = "/tmp/" + common.Title + "/test" + strconv.Itoa(int(time.Now().UnixNano()))
}

func kv() *bitcask.DB {
	Start()
	return db
}

func Start() {
	once.Do(func() {
		setTarget()
		if err := os.MkdirAll(target, 0o755); err != nil {
			log.Fatal().Err(err).Msg("error creating data directory")
		}
		db = bitcask.OpenDB(target)
		for _, store := range stores {
			if err := db.Init(store); err != nil {
				log.Fatal().Err(err).Str("store", store).Msg("error initializing store")
			}
		}
	})
}

func Close() {
	if err := db.SyncAndCloseAll(); err != nil {
		log.Warn().Err(err).Msg("error syncing and closing db")
	}
}
