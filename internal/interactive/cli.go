package interactive

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
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

func validate(input string) error {
	if len(strings.TrimSpace(input)) < 1 {
		return errors.New("no input detected")
	}
	return nil
}

type Selection struct {
	Bridge string
	Action string
	Target struct {
		Type string
		Name string
	}
}

type pool struct {
	p *sync.Pool
}

var stringers = pool{p: &sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	}}}

func (p *pool) Get() *strings.Builder {
	return p.p.Get().(*strings.Builder)
}

func (p *pool) Put(sb *strings.Builder) {
	sb.Reset()
	p.p.Put(sb)
}

func (s *Selection) String() string {
	if s.Bridge == "" && s.Action == "" {
		return "~"
	}
	builder := stringers.Get()
	builder.WriteString(s.Bridge)
	if s.Action != "" {
		builder.WriteString("/")
		builder.WriteString(s.Action)
	}
	if s.Target.Type != "" {
		builder.WriteString("/")
		builder.WriteString(s.Target.Type)
		builder.WriteString("s")
	}
	if s.Target.Name != "" {
		builder.WriteString("/")
		builder.WriteString(s.Target.Name)
	}
	res := builder.String()
	stringers.Put(builder)
	return res
}

var sel = &Selection{}

func InteractiveAuth() string {
	passPrompt := tui.Prompt{
		Label:    "API Key (AKA user)",
		Validate: validate,
		Mask:     '',
	}

	for {
		key, perr := passPrompt.Run()

		if perr != nil {
			fmt.Println(tui.IconBad + perr.Error())
			continue
		}
		return key
	}
}

// Interpret is where we will actuall define our commands
func executor(cmd string) {
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
		if br, ok := ziggy.Lucifer.Bridges[args[1]]; !ok {
			println("invalid bridge: " + args[1])
		} else {
			sel.Bridge = args[1]
			log.Info().Str("host", br.Host).Int("lights", len(br.HueLights)).Msg("switched to bridge: " + sel.Bridge)
		}
	case "debug":
		levelsdebug := map[string]zerolog.Level{"info": zerolog.InfoLevel, "debug": zerolog.DebugLevel, "trace": zerolog.TraceLevel}
		debuglevels := map[zerolog.Level]string{zerolog.InfoLevel: "info", zerolog.DebugLevel: "debug", zerolog.TraceLevel: "trace"}
		if len(args) < 2 {
			println("current debug level: " + debuglevels[log.GetLevel()])
		}
		if newlevel, ok := levelsdebug[args[1]]; ok {
			zerolog.SetGlobalLevel(newlevel)
		} else {
			println("invalid argument: " + args[1])
		}
	case "help":
		if len(args) < 2 {
			getHelp("meta")
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
		bcmd, ok := bridgeCMD[args[0]]
		if !ok {
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
					err := bcmd(brj, args[1:])
					if err != nil {
						log.Error().Err(err).Msg("bridge command failed")
					}
				}(br)
			}
		} else {
			err := bcmd(br, args[1:])
			if err != nil {
				log.Error().Err(err).Msg("error executing command")
			}
		}
	}
}

func getHelp(target string) {
	fmt.Printf("pRaNkeD! (help not available just yet.)\n")
	/*
		var lines []string

		lines = append(lines, "help: "+target)

		switch target {

		case "meta":
			var list string
			for _, cmd := range cmds {
				list = list + cmd + ", "
			}

			fmt.Println("Enabled commands: ")
			list = strings.TrimSuffix(list, ", ")
			fmt.Println(list)
			fmt.Println()

		default:
			log.Error().Msg("Help entry not found!")
			fmt.Println()
		}
	*/
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

var selectedBridge = ""

const bulb = ``

func getHist() []string {
	return []string{}
}

func StartCLI() {
	log = config.GetLogger()
	processBridges()
	grpmap, err := getGroupMap()
	if err != nil {
		log.Fatal().Err(err).Msg("error getting group map")
	}
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
					for brid, _ := range ziggy.Lucifer.Bridges {
						sel.Bridge = brid
					}
				}
				return fmt.Sprintf("ziggs[%s] %s ", sel.String(), bulb), true
			}),
		cli.OptionTitle("ziggs"),
		cli.OptionCompletionOnDown(),
	)

	prompt.Run()
}
