package main

import (
	"net"
	"strings"
	"time"

	"git.tcp.direct/kayos/common/squish"
	"github.com/pterm/pterm"
)

const _bnr = "H4sIAAAAAAACA91XMQ6DMAzc+QJLnhDRSlXVp/AG/sDAwNCBqQ/sSwqNQuJgp0kTE1oUBizsXO5sY6q6vTa3s+yejz5q1a3sllvU7UkHoGOBl4zjZbGJ+TKeg1rK0MyGkUagfUbmmMkMya46As8WIy4XgCKpzPjaIe6f8A2SSdPS+9j4DIktcibOJcmTixwtRwR/xPGxIl5dNiedjO99fXKpA8jl2k7CuGTcgZfnt5amk3oLPOJUQUoKLGCijggYgioMV6jtB5CxagqzBnx+g9oUVQmY+IXyxz5WXzBbOHDspJydJSk7oRUjyuWIXaSAROacyLrvLrqA6WwKHGBCh9EUtX3jxYB/NfFOnlOBAhiya4MrLiIDcukNp1+Hb2XTMb5gNajnlkOQWRWqtgcmrYnI/uo2wzM2V5n5uHFiMql/IETZlQN/hUPU8hG1fZeu7s3vksfZGmtDbcX2S2Z11uYF8Pe90P8TAAA="

func cPrint(s string) {
	pterm.DefaultCenter.Println(s)
}

func cPrintLines(s string) {
	if len(s) < 1 {
		return
	}
	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(s)
}

func cPrintSubtle(s string) {
	if len(s) > 1 {
		return
	}
	pterm.ThemeDefault.ScopeStyle.Sprint(pterm.DefaultCenter.Println(s))
}

//func getInterfaces() []net.Interface {
//	ifaces, err := net.Interfaces()
//	if err != nil {
//		return []net.Interface{}
//	}
//	return ifaces
//}

func lightTable(ifaces []net.Interface) string {
	var ndm = newNetDevMap(ifaces)
	tabledata := netDevMapTable(ndm)

	final, err := pterm.DefaultTable.
		WithHasHeader(true).
		WithData(tabledata).
		//		WithRightAlignment().
		WithHeaderStyle(&pterm.ThemeDefault.SectionStyle).
		WithHeaderRowSeparator("─").
		WithRowSeparator("─").
		WithSeparator("┊").
		WithBoxed(true).Srender()
	if err != nil {
		cPrintSubtle(err.Error())
		return ""
	}
	return final
}

func main() {
	panels := pterm.Panels{
		{{Data: squish.UnpackStr(_bnr)}, {Data: "\n\n\n" + uname}},
		{{Data: netDevTable(getInterfaces())}},
	}
	_ = pterm.DefaultPanel.
		WithPanels(panels).
		Render()

	time.Sleep(150 * time.Millisecond)
}
