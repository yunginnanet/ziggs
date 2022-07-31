package config

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"git.tcp.direct/kayos/ziggs/internal/common"
)

var (
	// CurrentLogFile is used for accessing the location of the currently used log file across packages.
	CurrentLogFile string
	logFile        *os.File
	LogDir         string
	logger         zerolog.Logger
	started        bool
)

// StartLogger instantiates an instance of our zerolog loggger so we can hook it in our main package.
// While this does return a logger, it should not be used for additional retrievals of the logger. Use GetLogger()
func StartLogger() *zerolog.Logger {
	LogDir = Snek.GetString("logger.directory")
	if err := os.MkdirAll(LogDir, 0o755); err != nil {
		println("cannot create log directory: " + LogDir + "(" + err.Error() + ")")
		os.Exit(1)
	}

	tnow := common.Title

	if Snek.GetBool("logger.use_date_filename") {
		tnow = strings.ReplaceAll(time.Now().Format(time.RFC822), " ", "_")
		tnow = strings.ReplaceAll(tnow, ":", "-")
	}

	CurrentLogFile = LogDir + tnow + ".log"

	if logFile, err = os.OpenFile(CurrentLogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666); err != nil {
		println("cannot create log file: " + err.Error())
		os.Exit(1)
	}

	defer func() {
		started = true
	}()
	multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{NoColor: NoColor, Out: os.Stdout}, logFile)
	logger = zerolog.New(multi).With().Timestamp().Logger()
	return &logger
}

// GetLogger retrieves our global logger object
func GetLogger() *zerolog.Logger {
	for !started {
		time.Sleep(10 * time.Millisecond)
	}
	return &logger
}
