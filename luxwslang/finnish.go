package luxwslang

import (
	"strings"
)

// Finnish language terminology.
var Finnish = &Terminology{
	ID:   "fi",
	Name: "Suomi",

	timestampFormat:      "02.01.06 15:04:05",
	timestampShortFormat: "02.01.06 15:04",

	NavInformation:  "Informaatio",
	NavTemperatures: "Lämpötilat",
	NavElapsedTimes: "Käyntiajat",
	NavInputs:       "Tilat sisäänmeno",
	NavOutputs:      "Tilat ulostulo",
	NavHeatQuantity: "Kalorimetri",
	NavEnergyInput:  "Power Consumption", // TODO
	NavErrorMemory:  "Häiriöloki",
	NavSwitchOffs:   "Pysähtymistieto",

	NavOpHours: "Käyttötunnit",
	HoursImpulsesFn: func(s string) bool {
		return strings.HasPrefix(s, "impulse") || strings.HasPrefix(s, "Impulse")
	},

	NavSystemStatus:       "Laitetiedot",
	StatusType:            "Lämpöpumpun tyyppi",
	StatusSoftwareVersion: "Ohjelmaversio",
	StatusOperationMode:   "Toimintatila",
	OperationModeMapping: map[string]float64{
		// TODO use finnish names
		// lower case!
		"off":        OpModeIDOff,
		"heating":    OpModeIDHeating,
		"evu":        OpModeIDEVU,
		"dhw":        OpModeIDDHW,
		"defrosting": OpModeIDDefrosting,
	},
	StatusPowerConsumption: "Kapasiteetti",
	StatusHeatingCapacity:  "Heating capacity", // might be the same as "actual capacity" // TODO use finnish names
	StatusDefrostDemand:    "Defrost demand",   // TODO use finnish names
	StatusLastDefrost:      "last defrost",     // TODO use finnish names

	BoolFalse: "Pois",
	BoolTrue:  "On",
}
