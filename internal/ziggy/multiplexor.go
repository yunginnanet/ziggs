package ziggy

import (
	"fmt"

	"github.com/amimof/huego"
)

// Multiplex is all of the lights (all of the lights).
// I'll see myself out.
type Multiplex struct {
	bridges []*Bridge
}

func GetGroupMap() (map[string]*huego.Group, error) {
	var groupmap = make(map[string]*huego.Group)
	for _, br := range Lucifer.Bridges {
		gs, err := br.GetGroups()
		if err != nil {
			return nil, err
		}
		for _, g := range gs {
			grp, gerr := br.GetGroup(g.ID)
			if gerr != nil {
				log.Warn().Msgf("[%s] %v", g.Name, gerr)
				continue
			}
			var count = 1
			groupName := g.Name
			for _, ok := groupmap[groupName]; ok; _, ok = groupmap[groupName] {
				groupName = fmt.Sprintf("%s_%d", g.Name, count)
			}
			groupmap[groupName] = grp
		}

	}
	return groupmap, nil
}

func GetLightMap() map[string]*huego.Light {
	var lightMap = make(map[string]*huego.Light)
	for _, l := range Lucifer.Lights {
		realLight, err := l.GetPtr()
		if err != nil {
			l.Log().Warn().Err(err).Msg("failed to get light pointer")
			continue
		}
		lightMap[realLight.Name] = realLight
	}
	return lightMap
}
