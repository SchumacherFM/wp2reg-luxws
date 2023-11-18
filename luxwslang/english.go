package luxwslang

import "strings"

// English language terminology.
var English = &Terminology{
	ID:   "en",
	Name: "English",

	timestampFormat: "02.01.06 15:04:05",

	NavInformation:  "information",
	NavTemperatures: "temperatures",
	NavElapsedTimes: "elapsed times",
	NavInputs:       "inputs",
	NavOutputs:      "outputs",
	NavHeatQuantity: "heat quantity",
	NavErrorMemory:  "error memory",
	NavSwitchOffs:   "switch offs",

	NavOpHours: "operating hours",
	HoursImpulsesFn: func(s string) bool {
		return strings.HasPrefix(s, "impulse") || strings.HasPrefix(s, "Impulse")
	},

	NavSystemStatus:       "system status",
	StatusType:            "type of heat pump",
	StatusSoftwareVersion: "software version",
	StatusOperationMode:   "operation mode",
	StatusPowerOutput:     "actual capacity",

	BoolFalse: "off",
	BoolTrue:  "on",
}
