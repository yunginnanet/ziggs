package cli

import (
	"github.com/amimof/huego"

	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

func printUpdateInfo(c *huego.Config) {
	log.Info().Msgf("Software version: %s", c.SwVersion)
	log.Info().Msgf("API version: %s", c.APIVersion)
	log.Info().Msgf("Datastore version: %s", c.DatastoreVersion)
	if len(c.SwUpdate2.LastInstall) > 0 {
		log.Info().Msgf("Last update: %v", c.SwUpdate2.LastInstall)
	}
	log.Info().Msgf("Auto install enabled: %t", c.SwUpdate2.AutoInstall.On)
	log.Info().Msgf("Auto install time: %v", c.SwUpdate2.AutoInstall.UpdateTime)
	log.Info().Msgf("Update state: %v", c.SwUpdate2.State)
}

func printNetworkInfo(c *huego.Config) {
	log.Info().Msgf("Zigbee channel: %d", c.ZigbeeChannel)
	log.Info().Msgf("IP address: %s", c.IPAddress)
	log.Info().Msgf("Netmask: %s", c.NetMask)
	log.Info().Msgf("Gateway: %s", c.Gateway)
	log.Info().Msgf("DHCP enabled: %t", c.Dhcp)
	log.Info().Msgf("Proxy address: %s", c.ProxyAddress)
	log.Info().Msgf("Proxy port: %d", c.ProxyPort)
}

func printRemoteServicesInfo(c *huego.Config) {
	log.Info().Msgf("WAN state: %s", c.InternetService.Internet)
	log.Info().Msgf("Remote access: %s", c.InternetService.RemoteAccess)
	log.Info().Msgf("NTP: %s", c.InternetService.Time)
	log.Info().Msgf("Update server: %s", c.InternetService.SwUpdate)
}

func printPortalInfo(c *huego.Config) {
	log.Info().Msgf("Portal connection: %s", c.PortalState.Communication)
	log.Info().Msgf("Portal signed on: %t", c.PortalState.SignedOn)
	log.Info().Msgf("Portal I/O: %t/%t", c.PortalState.Incoming, c.PortalState.Outgoing)
}

func cmdInfo(br *ziggy.Bridge, args []string) error {
	return printBridgeInfo(br)
}

func printBridgeInfo(br *ziggy.Bridge) error {
	c, err := br.GetConfig()
	if err != nil {
		return err
	}
	log.Info().Msgf("Name: %s", c.Name)
	log.Info().Msgf("ID: %s", c.BridgeID)
	log.Info().Msgf("MAC: %s", c.Mac)
	log.Info().Msgf("Model: %s", c.ModelID)
	println()
	printUpdateInfo(c)
	println()
	printNetworkInfo(c)
	println()
	printRemoteServicesInfo(c)
	println()
	printPortalInfo(c)
	println()
	log.Info().Msgf("Local Time: %s", c.LocalTime)
	log.Info().Msgf("Link Button Enabled: %t", c.LinkButton)
	return nil
}
