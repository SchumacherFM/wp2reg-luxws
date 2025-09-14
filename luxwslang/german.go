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
	OperationModeDefault:  "off",
	OperationModeMapping: map[string]float64{
		// lower case!
		"":       OpModeIDOff,
		"off":    OpModeIDOff,
		"heizen": OpModeIDHeating,
		"evu":    OpModeIDEVU,
		"ww":     OpModeIDDHW,
		"abt":    OpModeIDDefrosting,
	},
	StatusPowerConsumption: "Eingesetzte Energie",
	StatusHeatingCapacity:  "Heizleistung Ist",
	StatusDefrostDemand:    "Abtaubedarf",
	StatusLastDefrost:      "Letzte Abt.",
	BoolFalse:              "Aus",
	BoolTrue:               "Ein",
}
