package interactive

import (
	cli "git.tcp.direct/Mirrors/go-prompt"
)

func completer(in cli.Document) []cli.Suggest {
	c := in.CurrentLine()
	if c == "" {
		return []cli.Suggest{}
	}
	// args := strings.Fields(c)
	var set []cli.Suggest
	for command, subcommand := range suggestions {
		head := in.CursorPositionCol()
		if head > len(command.Text) {
			head = len(command.Text)
		}
		tmpl := &cli.Document{Text: command.Text}
		one := tmpl.GetWordAfterCursor()
		if one != "" && one != command.Text {
			set = append(set, cli.Suggest{Text: tmpl.Text})
		}
		if one == "use" && len(subcommand) > 1 {
			set = append(set, cli.Suggest{Text: tmpl.Text})
		}
		for _, a := range subcommand {
			if head > len(tmpl.Text+a.Text)+1 {
				continue
			}
			tmpl = &cli.Document{Text: tmpl.Text + " " + a.Text}
			two := tmpl.GetWordAfterCursorWithSpace()
			if two != "" {
				set = append(set, cli.Suggest{Text: tmpl.Text})
			}
		}
	}
	return cli.FilterHasPrefix(set, c, false)
}
