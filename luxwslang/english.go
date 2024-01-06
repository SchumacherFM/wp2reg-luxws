package luxwslang

import "strings"

// English language terminology.
var English = &Terminology{
	ID:   "en",
	Name: "English",

	timestampFormat:      "02.01.06 15:04:05",
	timestampShortFormat: "02.01.06 15:04",

	NavInformation:  "information",
	NavTemperatures: "temperatures",
	NavElapsedTimes: "elapsed times",
	NavInputs:       "inputs",
	NavOutputs:      "outputs",
	NavHeatQuantity: "heat quantity",
	NavEnergyInput:  "energy input",
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
	StatusPowerOutput:     "actual capacity",  // might be the same as "Heating capacity"
	StatusHeatingCapacity: "Heating capacity", // might be the same as "actual capacity"
	StatusDefrostDemand:   "Defrost demand",
	StatusLastDefrost:     "last defrost",
	BoolFalse:             "off",
	BoolTrue:              "on",
}
