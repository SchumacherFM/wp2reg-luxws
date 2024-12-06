package luxwslang

import (
	"strings"
)

// Dutch language terminology.
var Dutch = &Terminology{
	ID:   "nl",
	Name: "Nederlands",

	timestampFormat:      "02.01.06 15:04:05",
	timestampShortFormat: "02.01.06 15:04",

	NavInformation:  "Informatie",
	NavTemperatures: "Temperaturen",
	NavElapsedTimes: "Aflooptijden",
	NavInputs:       "Ingangen",
	NavOutputs:      "Uitgangen",
	NavHeatQuantity: "Energie",
	NavEnergyInput:  "energy input", // todo Cyrill
	NavErrorMemory:  "Storingsbuffer",
	NavSwitchOffs:   "Afschakelingen",

	NavOpHours: "Bedrijfsuren",
	HoursImpulsesFn: func(s string) bool {
		return strings.HasPrefix(s, "impulse") || strings.HasPrefix(s, "Impulse")
	},

	NavSystemStatus:        "Installatiestatus",
	StatusType:             "Warmtepomp Type",
	StatusSoftwareVersion:  "Softwareversie",
	StatusOperationMode:    "Bedrijfstoestand",
	StatusPowerConsumption: "Vermogen",
	StatusHeatingCapacity:  "Heating capacity", // TODO correct translation
	StatusDefrostDemand:    "Defrost demand",   // TODO correct translation
	StatusLastDefrost:      "last defrost",     // TODO correct translation
	BoolFalse:              "Uit",
	BoolTrue:               "Aan",
}
