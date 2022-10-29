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

func (c *Bridge) FindGroup(input string) (light *huego.Group, err error) {
	var groupID int
	if groupID, err = strconv.Atoi(input); err != nil {
		targ, ok := GetGroupMap()[input]
		if !ok {
			return nil, fmt.Errorf("unable to resolve light ID from input: %s", input)
		}
		return targ, nil
	}
	return c.GetGroup(groupID)
}
