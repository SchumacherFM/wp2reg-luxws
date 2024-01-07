package luxwslang

import "strings"

// German language terminology.
var German = &Terminology{
	ID:   "de",
	Name: "Deutsch",

	timestampFormat:      "02.01.06 15:04:05",
	timestampShortFormat: "02.01.06 15:04",

	NavInformation:  "Informationen",
	NavTemperatures: "Temperaturen",
	NavElapsedTimes: "Ablaufzeiten",
	NavInputs:       "Eingänge",
	NavOutputs:      "Ausgänge",
	NavHeatQuantity: "Wärmemenge",
	NavEnergyInput:  "Eingesetzte Energie",
	NavErrorMemory:  "Fehlerspeicher",
	NavSwitchOffs:   "Abschaltungen",

	NavOpHours: "Betriebsstunden",
	HoursImpulsesFn: func(s string) bool {
		return strings.HasPrefix(s, "impulse") || strings.HasPrefix(s, "Impulse")
	},

	NavSystemStatus:       "Anlagenstatus",
	StatusType:            "Wärmepumpen Typ",
	StatusSoftwareVersion: "Softwarestand",
	StatusOperationMode:   "Betriebszustand",
	StatusPowerOutput:     "Leistung Ist",
	StatusHeatingCapacity: "Heating capacity", // TODO correct translation
	StatusDefrostDemand:   "Defrost demand",   // TODO correct translation
	StatusLastDefrost:     "last defrost",     // TODO correct translation
	BoolFalse:             "Aus",
	BoolTrue:              "Ein",
}
