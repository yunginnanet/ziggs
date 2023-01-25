package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cli "git.tcp.direct/Mirrors/go-prompt"
	tui "github.com/manifoldco/promptui"
	"github.com/rs/zerolog"

	"github.com/davecgh/go-spew/spew"

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

// Interpret is where we will actuall define our Commands
func executor(cmd string) {
	var status = 0
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("PANIC: %s", r)
		}
		if _, ok := noHist[cmd]; !ok && status == 0 {
			history = append(history, cmd)
			go saveHist()
		}
	}()
	cmd = strings.TrimSpace(cmd)
	var args = strings.Fields(cmd)
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

	case "debug":
		levelsdebug := map[string]zerolog.Level{"info": zerolog.InfoLevel, "debug": zerolog.DebugLevel, "trace": zerolog.TraceLevel}
		debuglevels := map[zerolog.Level]string{zerolog.InfoLevel: "info", zerolog.DebugLevel: "debug", zerolog.TraceLevel: "trace"}
		if len(args) < 2 {
			println("current debug level: " + debuglevels[log.GetLevel()])
			return
		}
		if newlevel, ok := levelsdebug[args[1]]; ok {
			zerolog.SetGlobalLevel(newlevel)
		} else {
			println("invalid argument: " + args[1])
		}
	case "help":
		if len(args) < 2 {
			getHelp("")
			return
		}
		getHelp(args[len(args)-1])
	case "clear":
		print("\033[H\033[2J")
	case "debugcli":
		if extraDebug {
			extraDebug = false
		} else {
			extraDebug = true
		}
		spew.Dump(suggestions)
	default:
		if len(args) == 0 {
			return
		}
		bcmd, ok := Commands[args[0]]
		if !ok {
			log.Error().Msg("invalid command: " + args[0])
			status = 1
			return
		}
		br, ok := ziggy.Lucifer.Bridges[sel.Bridge]
		if sel.Bridge == "" || !ok {
			q := tui.Select{
				Label:   "Send to all known bridges?",
				Items:   []string{"yes", "no"},
				Pointer: common.ZiggsPointer,
			}
			_, ch, _ := q.Run()
			if ch != "yes" {
				return
			}
			for _, br := range ziggy.Lucifer.Bridges {
				go func(brj *ziggy.Bridge) {
					err := bcmd.reactor(brj, args[1:])
					if err != nil {
						log.Error().Err(err).Msg("bridge command failed")
					}
				}(br)
			}
			return
		}

		err := bcmd.reactor(br, args[1:])
		if err != nil {
			log.Error().Err(err).Msg("error executing command")
			status = 1
		}
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
			history = append(history, xerox.Text())
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

func StartCLI() {
	log = config.GetLogger()
	processBridges()
	grpmap := ziggy.GetGroupMap()
	processGroups(grpmap)
	processLights()
	prompt = cli.New(
		executor,
		completer,
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
		cli.OptionTitle("ziggs"),
		// cli.OptionCompletionOnDown(),
	)

	prompt.Run()
}
