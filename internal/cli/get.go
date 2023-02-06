package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/amimof/huego"

	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

// cmdGet is used to get the state(s) of lights and groups. it outputs the data in tab-delimited JSON.
func cmdGet(bridge *ziggy.Bridge, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments")
	}

	var (
		groupMap     map[string]*ziggy.HueGroup
		lightMap     map[string]*ziggy.HueLight
		currentState *huego.State
		argHead      = -1
	)

	for range args {
		argHead++
		if len(args) <= argHead {
			break
		}
		log.Trace().Int("argHead", argHead).Msg(args[argHead])
		switch args[argHead] {
		case "group", "g":
			groupMap = ziggy.GetGroupMap()
			if len(args) <= argHead-1 {
				return errors.New("no group specified")
			}
			argHead++
			g, ok := groupMap[strings.TrimSpace(args[argHead])]
			if !ok {
				return fmt.Errorf("group %s not found (argHead: %d)", args[argHead], argHead)
			}
			log.Trace().Str("group", g.Name).Msgf("found group %s via args[%d]",
				args[argHead], argHead,
			)
			currentState = g.State
		case "light", "l":
			lightMap = ziggy.GetLightMap()
			if len(args) <= argHead-1 {
				return errors.New("no light specified")
			}
			argHead++
			l, ok := lightMap[strings.TrimSpace(args[argHead])]
			if !ok {
				return fmt.Errorf("light %s not found (argHead: %d)", args[argHead], argHead)
			}
			if extraDebug {
				log.Trace().Str("group", l.Name).Msgf("found light %s via args[%d]",
					args[argHead], argHead)
			}
			currentState = l.State
		}
	}

	if currentState == nil {
		return errors.New("no state found")
	}

	data, err := json.MarshalIndent(currentState, "", "\t")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = io.Copy(os.Stdout, bytes.NewReader(data))
	return err
}
