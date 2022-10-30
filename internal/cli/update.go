package cli

import (
	"context"
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
		log.Warn().Msg("bridge is already set to check for updates")
	}
	if resp, err = br.UpdateConfig(&huego.Config{SwUpdate2: huego.SwUpdate2{CheckForUpdate: true}}); err == nil {
		log.Info().Msgf("response: %v", resp)
	} else {
		log.Warn().Msgf("failed to issue update command: %v", err)
	}

	ctx, cancel := watchUpdateStatus(br, 5*time.Minute)
	defer cancel()
	<-ctx.Done()

	log.Trace().Msg("retrieving bridge config...")
	var cNew *huego.Config
	cNew, err = br.GetConfig()
	if err != nil {
		return err
	}
	log.Trace().Msgf("new bridge update state:\n%s", spew.Sdump(cNew.SwUpdate2))
	log.Info().Msgf("New software version: %s", c.SwVersion)
	log.Info().Msgf("New update state: %v", c.SwUpdate2.State)
	return err
}

func watchUpdateStatus(br *ziggy.Bridge, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	stream := streamUpdateStatus(br, ctx, cancel)
	last := ""
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Trace().Msg("context for update status watcher done")
				return
			case state := <-stream:
				if state == last {
					continue
				}
				log.Info().Msgf("Update state: %s", state)
				last = state
				switch state {
				case "noupdates":
					log.Info().Msg("no updates available")
					cancel()
				case "allreadytoinstall":
					log.Info().Msg("all updates ready to install")
					log.Info().Msg("installing updates...")
					if _, err := br.UpdateConfig(&huego.Config{SwUpdate2: huego.SwUpdate2{Install: true}}); err != nil {
						log.Error().Err(err).Msg("error sending install command")
						cancel()
					}
				case "downloadready":
					log.Info().Msg("update ready to download")
					log.Info().Msg("downloading updates...")
					if _, err := br.UpdateConfig(&huego.Config{SwUpdate2: huego.SwUpdate2{Install: true}}); err != nil {
						log.Error().Err(err).Msg("error sending download command")
						cancel()
					}
				case "downloaded":
					log.Info().Msg("update downloaded...")
				case "updating":
					log.Debug().Msg("update in progress...")
				case "transfering":
					log.Info().Msg("update transfering...")
				case "idle":
					log.Info().Msg("update complete!")
					cancel()
				}
			}
		}
	}()

	return ctx, cancel
}

func streamUpdateStatus(br *ziggy.Bridge, ctx context.Context, cancel context.CancelFunc) chan string {
	ch := make(chan string)
	var errCount int
	go func() {
		defer log.Trace().Msg("streamUpdateStatus exiting")
		for {
			time.Sleep(1 * time.Second)
			if errCount > 5 {
				cancel()
				log.Fatal().Msg("too many errors, aborting")
			}
			select {
			case <-ctx.Done():
				cancel()
				return
			default:
				c, err := br.GetConfig()
				if err != nil {
					log.Error().Err(err).Msg("error retrieving bridge config")
					errCount++
					time.Sleep(1 * time.Second)
					continue
				}
				log.Trace().Msgf("bridge update state: %s", c.SwUpdate2.State)
				ch <- c.SwUpdate2.State
			}
		}
	}()
	return ch
}
