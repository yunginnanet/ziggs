package ziggy

import "strconv"

// Multiplex is all of the lights (all of the lights).
// I'll see myself out.
type Multiplex struct {
	bridges []*Bridge
}

var (
	lightMap    map[string]*HueLight
	groupMap    map[string]*HueGroup
	sensorMap   map[string]*HueSensor
	sceneMap    map[string]*HueScene
	needsUpdate = 4
)

func NeedsUpdate() {
	needsUpdate = 4
}

func GetLightMap() map[string]*HueLight {
	if needsUpdate == 0 {
		return lightMap
	}

	defer func() {
		needsUpdate--
	}()

	lightMap = make(map[string]*HueLight)
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
			lightMap[l.Name] = &HueLight{Light: light, controller: c}
		}
	}
	return lightMap
}

func GetGroupMap() map[string]*HueGroup {
	if needsUpdate == 0 {
		return groupMap
	}

	defer func() {
		needsUpdate--
	}()

	groupMap = make(map[string]*HueGroup)
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
			hg := &HueGroup{Group: group, controller: c}
			groupMap[g.Name] = hg
			groupMap[strconv.Itoa(g.ID)] = hg
		}
	}
	return groupMap
}

func GetSensorMap() map[string]*HueSensor {
	if needsUpdate == 0 {
		return sensorMap
	}

	defer func() {
		needsUpdate--
	}()

	sensorMap = make(map[string]*HueSensor)
	for _, c := range Lucifer.Bridges {
		ss, err := c.GetSensors()
		if err != nil {
			log.Warn().Msgf("error getting groups on bridge %s: %v", c.ID, err)
			continue
		}
		for _, s := range ss {
			sensor, gerr := c.GetSensor(s.ID)
			if gerr != nil {
				log.Warn().Msgf("failed to get pointer for sensor %s on bridge %s: %v", s.Name, c.ID, gerr)
				continue
			}
			if _, ok := sensorMap[s.Name]; ok {
				log.Warn().Msgf("duplicate sensor name %s on bridge %s - please rename", s.Name, c.ID)
				continue
			}
			sensorMap[s.Name] = &HueSensor{Sensor: sensor, controller: c}
		}
	}
	return sensorMap
}

func GetSceneMap() map[string]*HueScene {
	if needsUpdate == 0 {
		return sceneMap
	}

	defer func() {
		needsUpdate--
	}()

	sceneMap = make(map[string]*HueScene)
	for _, c := range Lucifer.Bridges {
		scs, err := c.GetScenes()
		if err != nil {
			log.Warn().Msgf("error getting groups on bridge %s: %v", c.ID, err)
			continue
		}
		for _, s := range scs {
			group, gerr := c.GetScene(s.ID)
			if gerr != nil {
				log.Warn().Msgf("failed to get pointer for scene %s on bridge %s: %v", s.Name, c.ID, gerr)
				continue
			}
			if _, ok := sceneMap[s.Name]; !ok {
				sceneMap[s.Name] = &HueScene{Scene: group, controller: c}
				continue
			}
			if _, ok := sceneMap[s.Name+"-2"]; ok {
				log.Warn().Msgf("duplicate scene name %s on bridge %s - please rename", s.Name, c.ID)
				continue
			}
		}
	}
	return sceneMap
}
