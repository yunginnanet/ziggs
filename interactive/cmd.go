package interactive

import (
	"fmt"
	"os"
	"strings"
	"time"

	cli "git.tcp.direct/Mirrors/go-prompt"
	. "github.com/logrusorgru/aurora"
	"github.com/rs/zerolog"

	"git.tcp.direct/kayos/ziggs/config"
	"git.tcp.direct/kayos/ziggs/lights"
)

var log *zerolog.Logger

var suggestions = []cli.Suggest{
	{"light list",
		"List known lights"},
}

func completer(in cli.Document) []cli.Suggest {
	c := in.CurrentLineBeforeCursor()
	if c == "" {
		return []cli.Suggest{}
	}
	return cli.FilterHasPrefix(suggestions, c, true)
}

// Interpret is where we will actuall define our commands
func executor(cmd string) {
	cmd = strings.TrimSpace(cmd)

	var args []string
	args = strings.Split(cmd, " ")

	getArgs := func(args []string) string {
		var ret string
		for i, a := range args {
			if i == 0 {
				ret = a
				continue
			}
			ret = ret + " " + a
			if i != len(args)-1 {
				ret = ret + " "
			}
		}
		return ret
	}

	switch args[0] {

	case "with":
		if len(args) <= 2 {
			return
		}
		b, ok := bridges[args[1]]
		if !ok {
			log.Error().Msg("unknown target")
			return
		}
		bcmd, ok := bridgeCMD[args[2]]
		if !ok {
			log.Error().Msg("unknown command")
			return
		}
		bcmd(b)

	case "quit", "exit":
		os.Exit(0)

	case "debug":
		if zerolog.GlobalLevel() == zerolog.InfoLevel {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
			log.Debug().
				Msg("Debug mode turned " + Sprintf(BrightGreen("ON\n").Bold()))
			return
		}
		log.Info().Msg("Debug mode turned " + Sprintf(BrightRed("OFF\n").Bold()))
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		return
	case "help":
		if len(args) < 2 {
			getHelp("meta")
			return
		}
		getHelp(getArgs(args))x
	case "clear":
		print("\033[H\033[2J")
		// termenv.ClearScreen()
	default:
		println()
	}

}

func getHelp(target string) {
	fmt.Printf("\npRaNkeD! (help not available just yet.)\n")
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

var bridges = make(map[string]*lights.Controller)
var bulbs = make(map[string]*lights.HueLight)

func cmdScan(br *lights.Controller) error {
	r, err := br.FindLights()
	if err != nil {
		return err
	}
	for resp, _ := range r.Success {
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

func cmdLights(br *lights.Controller) error {
	return nil
}

type reactor func(br *lights.Controller) error

var bridgeCMD = map[string]reactor{"scan": cmdScan, "lights": cmdLights}

func processBridges(Known []*lights.Controller) {
	for _, c := range Known {
		bridges[c.ID] = c
		for command, _ := range bridgeCMD {
			newsug := cli.Suggest{
				Text:        "bridge " + c.Bridge.Host + " " + command,
				Description: c.Host,
			}
			suggestions = append(suggestions, newsug)
		}
	}
}

func StartCLI(Known []*lights.Controller) {
	log = config.GetLogger()

	var hist []string
	processBridges(Known)

	p := cli.New(
		executor,
		completer,
		cli.OptionPrefix("ziggs"+"~ "),
		// cli.OptionPrefixBackgroundColor(cli.Black),
		cli.OptionPrefixTextColor(cli.Yellow),
		cli.OptionHistory(hist),
		cli.OptionSuggestionBGColor(cli.Black),
		cli.OptionSuggestionTextColor(cli.White),
		cli.OptionSelectedSuggestionBGColor(cli.Black),
		cli.OptionSelectedSuggestionTextColor(cli.Green),
		// prompt.OptionLivePrefix
		cli.OptionTitle("ziggs"),
	)

	p.Run()

}
