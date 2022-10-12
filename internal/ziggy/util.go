package ziggy

import (
	"fmt"
	"strconv"

	"github.com/amimof/huego"
)

func (c *Bridge) FindLight(input string) (light *huego.Light, err error) {
	var lightID int
	if lightID, err = strconv.Atoi(input); err != nil {
		targ, ok := GetLightMap()[input]
		if !ok {
			return nil, fmt.Errorf("unable to resolve light ID from input: %s", input)
		}
		return targ, nil
	}
	return c.GetLight(lightID)
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
