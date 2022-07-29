package interactive

import cli "git.tcp.direct/Mirrors/go-prompt"

func completer(in cli.Document) []cli.Suggest {
	c := in.CurrentLine()
	if c == "" {
		return []cli.Suggest{}
	}
	var set []cli.Suggest
	for command, arguments := range suggestions {
		head := in.CursorPositionCol()
		if head > len(command.Text) {
			head = len(command.Text)
		}
		tmpl := &cli.Document{Text: command.Text}
		one := tmpl.GetWordAfterCursor()
		if one != "" && one != "use" {
			set = append(set, cli.Suggest{Text: tmpl.Text})
		}
		if one == "use" && len(arguments) > 1 {
			set = append(set, cli.Suggest{Text: tmpl.Text})
		}
		for _, a := range arguments {
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
	return cli.FilterHasPrefix(set, c, true)
}
