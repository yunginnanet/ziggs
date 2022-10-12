package ziggy

import "github.com/amimof/huego"

// Multiplex is all of the lights (all of the lights).
// I'll see myself out.
type Multiplex struct {
	bridges []*Bridge
}

func GetLightMap() map[string]*huego.Light {
	var lightMap = make(map[string]*huego.Light)
	for _, c := range Lucifer.Bridges {
		ls, err := c.GetLights()
		if err != nil {
			log.Warn().Msgf("error getting lights on bridge %s: %v", c.ID, err)
			continue
		}
		for _, l := range ls {
			light, lerr := c.GetLight(l.ID)
			if lerr != nil {
				log.Warn().Msgf("failed to get pointer for light %s on bridge %s: %v", l.Name, c.ID, lerr)
				continue
			}
			if _, ok := lightMap[l.Name]; ok {
				log.Warn().Msgf("duplicate light name %s on bridge %s - please rename", l.Name, c.ID)
				continue
			}
			lightMap[l.Name] = light
		}
	}
	return lightMap
}

func GetGroupMap() map[string]*huego.Group {
	var groupMap = make(map[string]*huego.Group)
	for _, c := range Lucifer.Bridges {
		gs, err := c.GetGroups()
		if err != nil {
			log.Warn().Msgf("error getting groups on bridge %s: %v", c.ID, err)
			continue
		}
		for _, g := range gs {
			group, gerr := c.GetGroup(g.ID)
			if gerr != nil {
				log.Warn().Msgf("failed to get pointer for group %s on bridge %s: %v", g.Name, c.ID, gerr)
				continue
			}
			if _, ok := groupMap[g.Name]; ok {
				log.Warn().Msgf("duplicate group name %s on bridge %s - please rename", g.Name, c.ID)
				continue
			}
			groupMap[g.Name] = group
		}
	}
	return groupMap
}
