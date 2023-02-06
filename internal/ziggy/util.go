package ziggy

import (
	"fmt"
	"strconv"

	"github.com/amimof/huego"
)

func (c *Bridge) FindLight(input string) (light *HueLight, err error) {
	var lightID int
	if lightID, err = strconv.Atoi(input); err != nil {
		targ, ok := GetLightMap()[input]
		if !ok {
			return nil, fmt.Errorf("unable to resolve light ID from input: %s", input)
		}
		return targ, nil
	}
	l, err := c.GetLight(lightID)
	if err != nil {
		return nil, err
	}
	return &HueLight{Light: l, controller: c}, nil
}

func (c *Bridge) FindGroup(input string) (light *HueGroup, err error) {
	var groupID int
	if groupID, err = strconv.Atoi(input); err != nil {
		targ, ok := GetGroupMap()[input]
		if !ok {
			return nil, fmt.Errorf("unable to resolve light ID from input: %s", input)
		}
		return targ, nil
	}
	var hg *huego.Group
	if hg, err = c.GetGroup(groupID); err != nil {
		return nil, err
	}

	return &HueGroup{Group: hg, controller: c}, nil
}

func (hg *HueGroup) Scenes() ([]*HueScene, error) {
	scenes, err := hg.controller.GetScenes()
	if err != nil {
		return nil, err
	}
	var ret []*HueScene
	for _, s := range scenes {
		intID, err := strconv.Atoi(s.Group)
		if err != nil {
			log.Warn().Msgf("unable to parse group ID from scene %s: %v", s.Name, err)
		}
		if intID != hg.ID {
			continue
		}
		s, err := hg.controller.GetScene(s.ID)
		if err != nil {
			log.Warn().Msgf("unable to get scene pointer for scene %s: %v", s.Name, err)
			return nil, err
		}
		ret = append(ret, &HueScene{Scene: s, controller: hg.controller})
	}
	return ret, nil
}
