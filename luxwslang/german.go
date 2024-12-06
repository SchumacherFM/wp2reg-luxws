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
	OperationModeMapping: map[string]float64{
		// lower case!
		"off":    OpModeIDOff,
		"heizen": OpModeIDHeating,
		"evu":    OpModeIDEVU,
		"ww":     OpModeIDDHW,
		"abt":    OpModeIDDefrosting,
	},
	StatusPowerConsumption: "Leistung Ist",
	StatusHeatingCapacity:  "Heizleistung Ist", // TODO correct translation
	StatusDefrostDemand:    "Abtaubedarf",
	StatusLastDefrost:      "Letzte Abt.", // TODO correct translation
	BoolFalse:              "Aus",
	BoolTrue:               "Ein",
}
