package data

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"git.tcp.direct/tcp.direct/database/bitcask"
	"github.com/rs/zerolog/log"

	"git.tcp.direct/kayos/ziggs/internal/common"
)

var (
	stores       = []string{"macros", "users", "sequences"}
	isTest       = false
	once         = &sync.Once{}
	target       string
	db           *bitcask.DB
	testLocation string
)

func testMode() {
	isTest = true
}

func setTarget() {
	if !isTest {
		target = common.Home + "/.local/share/" + common.Title + "/"
	}
	testLocation = filepath.Join("/tmp", common.Title, strconv.FormatInt(time.Now().UnixNano(), 10))
	target = testLocation
}

func kv() *bitcask.DB {
	Start()
	return db
}

func startDB() {
	setTarget()
	if isTest {
		_ = os.RemoveAll(testLocation)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		log.Fatal().Err(err).Msg("error creating data directory")
	}
	db = bitcask.OpenDB(target)
	for _, store := range stores {
		if err := db.Init(store); err != nil {
			log.Fatal().Err(err).Str("store", store).Msg("error initializing store")
		}
	}
}

func Start() {
	if !isTest {
		once.Do(startDB)
		return
	}
	startDB()
}

func Close() {
	if err := db.SyncAndCloseAll(); err != nil {
		log.Warn().Err(err).Msg("error syncing and closing db")
	}
}
