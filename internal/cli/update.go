package cli

import (
	"fmt"
	"time"

	"github.com/amimof/huego"
	"github.com/davecgh/go-spew/spew"

	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

func cmdFirmwareUpdate(br *ziggy.Bridge, args []string) error {
	log.Trace().Msg("retrieving bridge config...")
	c, err := br.GetConfig()
	if err != nil {
		return err
	}
	printUpdateInfo(c)

	log.Info().Msg("attempting to trigger a firmware update...")
	log.Debug().Msgf("current bridge update state:\n%s", spew.Sdump(c.SwUpdate2))
	var resp *huego.Response
	if c.SwUpdate2.CheckForUpdate {
		return fmt.Errorf("bridge is already set to check for updates")
	}
	if resp, err = br.UpdateConfig(&huego.Config{SwUpdate2: huego.SwUpdate2{CheckForUpdate: true}}); err == nil {
		log.Info().Msgf("response: %v", resp)
	}
	log.Info().Msg("waiting for bridge to check for updates...")
	time.Sleep(5 * time.Second)
	log.Trace().Msg("retrieving bridge config...")
	var cNew *huego.Config
	cNew, err = br.GetConfig()
	if err != nil {
		return err
	}
	log.Debug().Msgf("new bridge update state:\n%s", spew.Sdump(cNew.SwUpdate2))
	log.Info().Msgf("New software version: %s", c.SwVersion)
	log.Info().Msgf("New update state: %v", c.SwUpdate2.State)
	return err
}
