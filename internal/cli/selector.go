package cli

import "git.tcp.direct/kayos/ziggs/internal/buffers"

func (s *Selection) String() string {
	if s.Bridge == "" && s.Action == "" {
		return "~"
	}
	builder := buffers.Stringers.Get()
	builder.WriteString(s.Bridge)
	if s.Action != "" {
		builder.WriteString("/")
		builder.WriteString(s.Action)
	}
	if s.Target.Type != "" {
		builder.WriteString("/")
		builder.WriteString(s.Target.Type)
		builder.WriteString("s")
	}
	if s.Target.Name != "" {
		builder.WriteString("/")
		builder.WriteString(s.Target.Name)
	}
	res := builder.String()
	buffers.Stringers.Put(builder)
	return res
}

var sel = &Selection{}

type Selection struct {
	Bridge string
	Action string
	Target struct {
		Type string
		Name string
	}
}
