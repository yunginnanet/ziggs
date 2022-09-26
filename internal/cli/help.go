package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

var tabber = tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)

func getHelp(target string) {
	if target != "" && target != "meta" {
		for _, su := range suggestions[0] {
			if strings.Contains(strings.ToLower(su.Text), strings.ToLower(target)) {
				println(su.Text + "\t" + su.Description)
			}
		}
		return
	}

	for _, su := range suggestions[0] {
		var desc string
		if su.inner == nil {
			desc = su.Description
		} else {
			desc = su.inner.description
			if su.inner.isAlias || su.isAlias {
				su.isAlias = true
				if extraDebug {
					log.Trace().Msgf("alias: %s", su.Text)
				}
			}
		}
		if su.isAlias {
			continue
		}
		if extraDebug {
			log.Trace().Interface("details", su).Send()
		}
		_, err := fmt.Fprintln(tabber, su.Text+"\t"+desc)
		if err != nil {
			panic(err.Error())
		}
	}
	tabber.Flush()
}
