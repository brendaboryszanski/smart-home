package domain

type DeviceType string

const (
	DeviceTypeLight      DeviceType = "light"
	DeviceTypePlug       DeviceType = "plug"
	DeviceTypeSwitch     DeviceType = "switch"
	DeviceTypeThermostat DeviceType = "thermostat"
	DeviceTypeSensor     DeviceType = "sensor"
	DeviceTypeOther      DeviceType = "other"
)

type Device struct {
	ID        string
	Name      string
	Type      DeviceType
	Category  string
	Online    bool
	Functions []DeviceFunction
}

type DeviceFunction struct {
	Code   string
	Type   string
	Values map[string]any
}

