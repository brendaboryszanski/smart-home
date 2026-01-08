package domain

type Action string

const (
	ActionTurnOn    Action = "turn_on"
	ActionTurnOff   Action = "turn_off"
	ActionSetLevel  Action = "set_level"
	ActionSetColor  Action = "set_color"
	ActionRunScene  Action = "run_scene"
	ActionGetStatus Action = "get_status"
	ActionUnknown   Action = "unknown"
)

// TextCommandPrefix is the marker used to indicate text commands (vs audio)
const TextCommandPrefix = "__TEXT__:"

type Command struct {
	Action     Action
	TargetName string
	TargetID   string
	TargetType TargetType
	Parameters map[string]any
	RawText    string
	Confidence float64
}

type TargetType string

const (
	TargetTypeDevice TargetType = "device"
	TargetTypeScene  TargetType = "scene"
)

