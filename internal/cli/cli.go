package cli

import (
	"bufio"
	"fmt"
	// "io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cli "git.tcp.direct/Mirrors/go-prompt"
	"github.com/rs/zerolog"

	"github.com/google/shlex"

	"git.tcp.direct/kayos/ziggs/internal/common"
	"git.tcp.direct/kayos/ziggs/internal/config"
	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

var (
	log        *zerolog.Logger
	prompt     *cli.Prompt
	extraDebug = false
)

var noHist = map[string]bool{"clear": true, "exit": true, "quit": true}

// Executor executes commands
func Executor(cmd string) {
	if log == nil {
		log = config.StartLogger()
	}

	log.Trace().Caller().Msg("getting readlock for suggestions")
	SuggestionMutex.RLock()
	defer SuggestionMutex.RUnlock()
	log.Trace().Caller().Msg("got readlock for suggestions")

	var status = 0
	defer func() {
		if r := recover(); r != nil {
			log.Error().Caller(3).Msgf("PANIC: %s", r)
		}
		if _, ok := noHist[cmd]; !ok && status == 0 {
			history = append(history, cmd)
			go saveHist()
		}
	}()

	// hacky bugfix
	cmd = strings.ReplaceAll(cmd, "#", "_POUNDSIGN_")
	args, err := shlex.Split(strings.TrimSpace(cmd))
	for i, arg := range args {
		args[i] = strings.ReplaceAll(arg, "_POUNDSIGN_", "#")
	}

	if err != nil {
		log.Error().Msgf("error parsing command: %s", err)
		status = 1
		return
	}
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "quit", "exit":
		os.Exit(0)
	case "use":
		if len(ziggy.Lucifer.Bridges) < 2 {
			return
		}
		if len(args) < 2 {
			println("use: use <bridge>")
			return
		}
		br, ok := ziggy.Lucifer.Bridges[args[1]]
		if !ok {
			log.Error().Msg("invalid bridge: " + args[1])
			status = 1
			return
		}
		sel.Bridge = args[1]
		log.Info().Str("host", br.Host).Int("lights", len(br.HueLights)).Msg("switched to bridge: " + sel.Bridge)
		return
	case "debug":
		levelsdebug := map[string]zerolog.Level{"info": zerolog.InfoLevel, "debug": zerolog.DebugLevel, "trace": zerolog.TraceLevel}
		debuglevels := map[zerolog.Level]string{zerolog.InfoLevel: "info", zerolog.DebugLevel: "debug", zerolog.TraceLevel: "trace"}
		if len(args) < 2 {
			log.Info().Msgf("current debug level: %s", debuglevels[log.GetLevel()])
			return
		}
		if newlevel, ok := levelsdebug[args[1]]; ok {
			zerolog.SetGlobalLevel(newlevel)
			nl := log.Level(newlevel)
			log = &nl
			return
		}
		if args[1] == "debugcli" || args[1] == "cli" {
			if extraDebug {
				extraDebug = false
				log.Info().Msg("disabled cli debug")
			} else {
				extraDebug = true
				/*				log.Info().Msgf("dumping suggestions")
								spew.Dump(suggestions)*/
				log.Info().Msg("enabled cli debug")
			}
			return
		}
		return
	case "help":
		if len(args) < 2 {
			getHelp("")
			return
		}
		getHelp(args[len(args)-1])
	case "clear":
		print("\033[H\033[2J")
		return
	default:
		if len(args) == 0 {
			return
		}

		complete := strings.Join(args, " ")
		log.Trace().Caller().Msgf("complete command: %s", complete)
		if strings.Contains(complete, "&&") {
			log.Warn().Caller().Msgf("found \"&&\" in command: %s, replacing with \";\"", complete)
			strings.ReplaceAll(complete, "&&", ";")
		}
		sep := strings.Split(complete, "&")
		log.Trace().Caller().Msgf("sep: %+s", sep)

		br, ok := ziggy.Lucifer.Bridges[sel.Bridge]
		if sel.Bridge == "" || !ok {
			for _, b := range ziggy.Lucifer.Bridges {
				br = ziggy.Lucifer.Bridges[b.Info.IPAddress]
				break
			}
		}

		wg := &sync.WaitGroup{}
		for _, cm := range sep {
			cm = strings.TrimSpace(cm)
			log.Trace().Caller().Msgf("executing command: %s", cm)
			wg.Add(1)
			go func(c string) {
				c = strings.TrimSpace(c)
				defer wg.Done()
				for _, synchro := range strings.Split(c, ";") {
					synchro = strings.TrimSpace(synchro)
					myArgs := strings.Split(synchro, " ")

					bcmd, myok := Commands[myArgs[0]]
					if !myok {
						log.Error().Msg("invalid command: " + myArgs[0])
						status = 1
						return
					}

					log.Trace().Caller().Msgf("selected bridge: %s", sel.Bridge)

					if e := bcmd.reactor(br, myArgs[1:]); e != nil {
						log.Error().Msgf("error executing command: %s", e)
						status = 1
						return
					}
				}
			}(cm)
		}

		wg.Wait()

	}
}

func cmdScan(br *ziggy.Bridge, args []string) error {
	r, err := br.FindLights()
	if err != nil {
		return err
	}
	for resp := range r.Success {
		log.Info().Msg(resp)
	}
	var count = 0
	timer := time.NewTimer(5 * time.Second)
	var newLights []string
loop:
	for {
		select {
		case <-timer.C:
			break loop
		default:
			newl, _ := br.GetNewLights()
			if len(newl.Lights) <= count {
				time.Sleep(250 * time.Millisecond)
				print(".")
				continue
			}
			count = len(newl.Lights)
			timer.Reset(5 * time.Second)
			newLights = append(newLights, newl.Lights...)
		}
	}
	for _, nl := range newLights {
		log.Info().Str("caller", nl).Msg("discovered light")
	}
	return nil
}

const bulb = ``

var (
	history    []string
	histLoaded bool
)

func loadHist() {
	var histMap = make(map[string]bool)
	pth, _ := filepath.Split(config.Filename)
	rb, _ := os.OpenFile(filepath.Join(pth, ".ziggs_history"), os.O_RDONLY, 0644)
	xerox := bufio.NewScanner(rb)
	for xerox.Scan() {
		_, ok := histMap[strings.TrimSpace(xerox.Text())]
		switch {
		case strings.TrimSpace(xerox.Text()) == "":
			continue
		case ok:
			continue
		default:
			histMap[strings.TrimSpace(xerox.Text())] = true
			history = append(history, strings.ReplaceAll(xerox.Text(), "_POUNDSIGN_", "#"))
		}
	}
	histLoaded = true
}

func getHist() []string {
	if !histLoaded {
		loadHist()
	}
	return history
}

func saveHist() {
	pth, _ := filepath.Split(config.Filename)
	_ = os.WriteFile(filepath.Join(pth, ".ziggs_history"), []byte(strings.Join(history, "\n")), 0644)
}

// func StartCLI(r io.Reader, w io.Writer) {
func StartCLI() {
	ct, _ := common.Version()
	// cli.NewStdoutWriter().
	prompt = cli.New(
		Executor,
		completer,
		//		cli.OptionWriter(w),
		//		cli.Op(r),
		// cli.OptionPrefixBackgroundColor(cli.Black),
		cli.OptionPrefixTextColor(cli.Yellow),
		cli.OptionHistory(getHist()),
		cli.OptionSuggestionBGColor(cli.Black),
		cli.OptionSuggestionTextColor(cli.White),
		cli.OptionSelectedSuggestionBGColor(cli.Black),
		cli.OptionSelectedSuggestionTextColor(cli.Green),
		cli.OptionLivePrefix(
			func() (prefix string, useLivePrefix bool) {
				if len(ziggy.Lucifer.Bridges) == 1 {
					for brid := range ziggy.Lucifer.Bridges {
						sel.Bridge = brid
					}
				}
				return fmt.Sprintf("ziggs[%s] %s ", sel.String(), bulb), true
			}),

		cli.OptionTitle("ziggs - built "+ct),
	)

	prompt.Run()
}
