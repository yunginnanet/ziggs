package ssh

import (
	"git.tcp.direct/kayos/common/squish"
	"github.com/charmbracelet/ssh"

	"git.tcp.direct/kayos/ziggs/internal/config"
)

func (s *Server) GetStaticKeys() map[string]ssh.PublicKey {
	if s.keys != nil {
		return s.keys
	}
	pubs := make(map[string]ssh.PublicKey)
	for _, key := range config.SSHPublicKeys {
		pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
		if err != nil {
			panic(err)
		}
		pubs[squish.B64e(pub.Marshal())] = pub
	}
	s.keys = pubs
	return pubs
}
