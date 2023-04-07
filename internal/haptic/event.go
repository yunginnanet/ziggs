package haptic

import "time"

type Color struct {
	Xy struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"xy"`
}

type Dynamics struct {
	Status     string  `json:"status,omitempty"`
	Speed      float64 `json:"speed,omitempty"`
	SpeedValid bool    `json:"speed_valid,omitempty"`
}

type Dimming struct {
	Brightness float64 `json:"brightness"`
}

type Owner struct {
	Rid   string `json:"rid"`
	Rtype string `json:"rtype"`
}

type Button struct {
	ButtonReport struct {
		Event   string    `json:"event"`
		Updated time.Time `json:"updated"`
	} `json:"button_report"`
	LastEvent string `json:"last_event"`
}

type ColorTemperature struct {
	Mirek      interface{} `json:"mirek"`
	MirekValid bool        `json:"mirek_valid"`
}

type Status struct {
	Active string `json:"active"`
}

type PowerState struct {
	BatteryLevel int    `json:"battery_level"`
	BatteryState string `json:"battery_state"`
}

type Temperature struct {
	Temperature       float64 `json:"temperature"`
	TemperatureReport struct {
		Changed     time.Time `json:"changed"`
		Temperature float64   `json:"temperature"`
	} `json:"temperature_report"`
	TemperatureValid bool `json:"temperature_valid"`
}

type WrappedEvent struct {
	Timestamp time.Time `json:"creationtime"`
	Id        string    `json:"id"`
	Type      string    `json:"type"`

	Event Event `json:"data"`
}

type Event struct {
	IdV1             string           `json:"id_v1"`
	Button           Button           `json:"button,omitempty"`
	Owner            Owner            `json:"owner,omitempty"`
	Dimming          Dimming          `json:"dimming,omitempty"`
	Dynamics         Dynamics         `json:"dynamics,omitempty"`
	Color            Color            `json:"color,omitempty"`
	ColorTemperature ColorTemperature `json:"color_temperature,omitempty"`
	Temperature      Temperature      `json:"temperature,omitempty"`
	PowerState       PowerState       `json:"power_state,omitempty"`
	Status           Status           `json:"status,omitempty"`
}
