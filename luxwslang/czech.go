package luxwslang

import (
	"strings"
)

// Czech language terminology.
var Czech = &Terminology{
	ID:   "cz",
	Name: "Česky",

	timestampFormat: "02.01.06 15:04:05",

	NavInformation:  "Informace",
	NavTemperatures: "Teploty",
	NavElapsedTimes: "Doby chodu",
	NavInputs:       "Vstupy",
	NavOutputs:      "Výstupy",
	NavHeatQuantity: "Teplo",
	NavEnergyInput:  "energy input", // todo Cyrill
	NavErrorMemory:  "Chybová paměť",
	NavSwitchOffs:   "Odepnutí",

	NavOpHours: "Provozní hodiny",

	HoursImpulsesFn: func(s string) bool {
		return strings.HasPrefix(s, "počet startů") || strings.HasPrefix(s, "Počet startů")
	},
	NavSystemStatus:       "Status zařízení",
	StatusType:            "Typ TČ",
	StatusSoftwareVersion: "Softwarová verze",
	StatusOperationMode:   "Provozní stav",
	StatusPowerOutput:     "Výkon",

	BoolFalse: "Vypnuto",
	BoolTrue:  "Zapnuto",
}
