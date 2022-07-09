package interactive

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	cli "git.tcp.direct/Mirrors/go-prompt"
	tui "github.com/manifoldco/promptui"
	"github.com/rs/zerolog"

	"git.tcp.direct/kayos/ziggs/common"
	"git.tcp.direct/kayos/ziggs/config"
	"git.tcp.direct/kayos/ziggs/ziggy"
)

var log *zerolog.Logger

func validate(input string) error {
	if len(strings.TrimSpace(input)) < 1 {
		return errors.New("no input detected")
	}
	return nil
}

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
	var args []string
	args = strings.Split(cmd, " ")
	switch args[0] {
	case "quit", "exit":
		os.Exit(0)
	case "use":
		if len(args) < 2 {
			println("use: use <bridge>")
			return
		}
		if br, ok := ziggy.Lucifer.Bridges[args[1]]; !ok {
			println("invalid bridge: " + args[1])
		} else {
			selectedBridge = args[1]
			log.Info().Str("host", br.Host).Int("lights", len(br.HueLights)).Msg("switched to bridge: " + selectedBridge)
		}
	case "debug":
		levelsdebug := map[string]zerolog.Level{"info": zerolog.InfoLevel, "debug": zerolog.DebugLevel, "trace": zerolog.TraceLevel}
		debuglevels := map[zerolog.Level]string{zerolog.InfoLevel: "info", zerolog.DebugLevel: "debug", zerolog.TraceLevel: "trace"}
		if len(args) < 2 {
			println("current debug level: " + debuglevels[log.GetLevel()])
		}
		if newlevel, ok := levelsdebug[args[1]]; ok {
			zerolog.SetGlobalLevel(newlevel)
		}
	case "help":
		if len(args) < 2 {
			getHelp("meta")
			return
		}
		getHelp(args[len(args)-1])
	case "clear":
		print("\033[H\033[2J")
	default:
		bcmd, ok := bridgeCMD[args[0]]
		if !ok {
			return
		}
		br, ok := ziggy.Lucifer.Bridges[selectedBridge]
		if selectedBridge == "" || !ok {
			prompt := tui.Select{
				Label:   "Send to all known bridges?",
				Items:   []string{"yes", "no"},
				Pointer: common.ZiggsPointer,
			}
			_, ch, _ := prompt.Run()
			if ch != "yes" {
				return
			}
			for _, br := range ziggy.Lucifer.Bridges {
				go bcmd(br, args[1:])
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
	return
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

func StartCLI() {
	log = config.GetLogger()

	var hist []string
	processBridges(ziggy.Lucifer.Bridges)

	/*	comphead := 0
		compmu := &sync.Mutex{}
		cleanSlate := func(b *cli.Buffer) {
			compmu.Lock()
			defer compmu.Lock()
			sugs := completer(*b.Document())
			if comphead > len(sugs)-2 {
				comphead = 0
				return
			}
			comphead++
			b.CursorLeft(len(b.Document().TextBeforeCursor()))
			b.InsertText(sugs[comphead].Text, true, true)
		}*/

	p := cli.New(
		executor,
		completer,
		// cli.OptionPrefixBackgroundColor(cli.Black),
		cli.OptionPrefixTextColor(cli.Yellow),
		cli.OptionHistory(hist),
		cli.OptionSuggestionBGColor(cli.Black),
		cli.OptionSuggestionTextColor(cli.White),
		cli.OptionSelectedSuggestionBGColor(cli.Black),
		cli.OptionSelectedSuggestionTextColor(cli.Green),
		cli.OptionLivePrefix(
			func() (prefix string, useLivePrefix bool) {
				sel := "~"
				if selectedBridge != "" {
					sel = selectedBridge
				}
				return fmt.Sprintf("ziggs[%s] %s ", sel, bulb), true
			}),
		cli.OptionTitle("ziggs"),
		/*		cli.OptionAddKeyBind(cli.KeyBind{
					Key: cli.Tab,
					Fn:  cleanSlate,
				}),
		*/cli.OptionCompletionOnDown(),
		/*		cli.OptionAddKeyBind(cli.KeyBind{
					Key: cli.Down,
					Fn:  cleanSlate,
				}),
		*/)

	p.Run()
}
