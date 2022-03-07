package interactive

import cli "git.tcp.direct/Mirrors/go-prompt"

func completer(in cli.Document) []cli.Suggest {
	c := in.CurrentLine()
	if c == "" {
		return []cli.Suggest{}
	}
	var set []cli.Suggest
	for command, arguments := range suggestions {
		head := in.CursorPosition
		if in.CursorPosition > len(command.Text) {
			head = len(command.Text)
		}
		tmpl := &cli.Document{Text: command.Text, CursorPosition: head}
		one := tmpl.GetWordAfterCursor()
		if one != "" {
			set = append(set, cli.Suggest{Text: tmpl.Text})
		}
		for _, a := range arguments {
			if in.CursorPositionCol() > len(tmpl.Text+a.Text)+1 {
				continue
			}
			head = in.CursorPosition
			tmpl = &cli.Document{Text: tmpl.Text + " " + a.Text, CursorPosition: head}
			two := tmpl.GetWordAfterCursorWithSpace()
			if two != "" {
				set = append(set, cli.Suggest{Text: tmpl.Text})
			}
		}
	}
	return cli.FilterHasPrefix(set, c, true)
}
