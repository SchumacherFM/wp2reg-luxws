package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/wp2reg-luxws/luxwsclient"
	"github.com/hansmi/wp2reg-luxws/luxwslang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func discardAllLogs(t *testing.T) {
	t.Helper()

	orig := log.Writer()

	t.Cleanup(func() {
		log.SetOutput(orig)
	})

	log.SetOutput(io.Discard)
}

type adapter struct {
	c *collector

	metricNames []string

	collect    func(ch chan<- prometheus.Metric) error
	collectErr error
}

func (a *adapter) Describe(ch chan<- *prometheus.Desc) {
	a.c.Describe(ch)
}

func (a *adapter) Collect(ch chan<- prometheus.Metric) {
	a.collectErr = a.collect(ch)
}

func (a *adapter) collectAndCompare(t *testing.T, want string, wantErr error) {
	t.Helper()

	if err := testutil.CollectAndCompare(a, strings.NewReader(want), a.metricNames...); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(wantErr, a.collectErr, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("Collection error diff (-want +got):\n%s", diff)
	}
}

func TestCollectWebSocketParts(t *testing.T) {
	c := newCollector(collectorOpts{
		terms: luxwslang.German,
		loc:   time.UTC,
	})

	for _, tc := range []struct {
		name       string
		fn         contentCollectFunc
		input      *luxwsclient.ContentRoot
		quirks     quirks
		want       string
		wantErr    error
		wantQuirks quirks
	}{
		{
			name: "info empty",
			fn:   c.collectInfo,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Anlagenstatus",
					},
				},
			},
			want: `
# HELP luxws_defrost Defrost demand in % and last defrost time
# TYPE luxws_defrost gauge
luxws_defrost{name="demand",unit=""} 0
luxws_defrost{name="last",unit="ts"} -6.21355968e+10
# HELP luxws_heat_capacity Heat Capacity
# TYPE luxws_heat_capacity gauge
luxws_heat_capacity{unit=""} 0
# HELP luxws_heat_quantity Heat quantity
# TYPE luxws_heat_quantity gauge
luxws_heat_quantity{unit=""} 0
# HELP luxws_info Controller information
# TYPE luxws_info gauge
luxws_info{hptype="",swversion=""} 1
# HELP luxws_operational_mode Operational mode
# TYPE luxws_operational_mode gauge
luxws_operational_mode{mode=""} 1
# HELP luxws_operational_mode_id Operational mode by ID
# TYPE luxws_operational_mode_id gauge
luxws_operational_mode_id{mode=""} 0
`,
		},
		{
			name: "info full",
			fn:   c.collectInfo,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Anlagenstatus",
						Items: luxwsclient.ContentItems{
							{Name: "Wärmepumpen Typ", Value: luxwsclient.String("typeA")},
							{Name: "Softwarestand", Value: luxwsclient.String("v1.2.3")},
							{Name: "Betriebszustand", Value: luxwsclient.String("running")},
							{Name: "Leistung Ist", Value: luxwsclient.String("999 kWh")},
							{Name: "Wärmepumpen Typ", Value: luxwsclient.String("typeB")},
						},
					},
				},
			},
			want: `
# HELP luxws_defrost Defrost demand in % and last defrost time
# TYPE luxws_defrost gauge
luxws_defrost{name="demand",unit=""} 0
luxws_defrost{name="last",unit="ts"} -6.21355968e+10
# HELP luxws_heat_capacity Heat Capacity
# TYPE luxws_heat_capacity gauge
luxws_heat_capacity{unit=""} 0
# HELP luxws_heat_quantity Heat quantity
# TYPE luxws_heat_quantity gauge
luxws_heat_quantity{unit="kWh"} 999
# HELP luxws_info Controller information
# TYPE luxws_info gauge
luxws_info{hptype="typeA, typeB",swversion="v1.2.3"} 1
# HELP luxws_operational_mode Operational mode
# TYPE luxws_operational_mode gauge
luxws_operational_mode{mode="running"} 1
# HELP luxws_operational_mode_id Operational mode by ID
# TYPE luxws_operational_mode_id gauge
luxws_operational_mode_id{mode="running"} 0
`,
		},
		{
			// https://github.com/hansmi/wp2reg-luxws/issues/11
			name: "info L2A model",
			fn:   c.collectInfo,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Anlagenstatus",
						Items: luxwsclient.ContentItems{
							{Name: "Wärmepumpen Typ", Value: luxwsclient.String("l2a")},
							{Name: "Softwarestand", Value: luxwsclient.String("v1.86.2")},
							{Name: "Betriebszustand", Value: luxwsclient.String("----")},
						},
					},
				},
			},
			want: `
# HELP luxws_defrost Defrost demand in % and last defrost time
# TYPE luxws_defrost gauge
luxws_defrost{name="demand",unit=""} 0
luxws_defrost{name="last",unit="ts"} -6.21355968e+10
# HELP luxws_heat_capacity Heat Capacity
# TYPE luxws_heat_capacity gauge
luxws_heat_capacity{unit=""} 0
# HELP luxws_heat_quantity Heat quantity
# TYPE luxws_heat_quantity gauge
luxws_heat_quantity{unit=""} 0
# HELP luxws_info Controller information
# TYPE luxws_info gauge
luxws_info{hptype="l2a",swversion="v1.86.2"} 1
# HELP luxws_operational_mode Operational mode
# TYPE luxws_operational_mode gauge
luxws_operational_mode{mode="----"} 1
# HELP luxws_operational_mode_id Operational mode by ID
# TYPE luxws_operational_mode_id gauge
luxws_operational_mode_id{mode="----"} 0
`,
			wantQuirks: quirks{
				missingSuppliedHeat: true,
			},
		},
		{
			name: "temperatures empty",
			fn:   c.collectTemperatures,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Temperaturen",
					},
				},
			},
			want: `
# HELP luxws_temperature Sensor temperature
# TYPE luxws_temperature gauge
luxws_temperature{name="",unit=""} 0
`,
		},
		{
			name: "temperatures full",
			fn:   c.collectTemperatures,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Temperaturen",
						Items: luxwsclient.ContentItems{
							{Name: "Wasser", Value: luxwsclient.String("10°C")},
							{Name: "Aussen", Value: luxwsclient.String("100°C")},
							{Name: "Etwas", Value: luxwsclient.String("1 K")},
						},
					},
				},
			},
			want: `
# HELP luxws_temperature Sensor temperature
# TYPE luxws_temperature gauge
luxws_temperature{name="Aussen",unit="degC"} 100
luxws_temperature{name="Etwas",unit="K"} 1
luxws_temperature{name="Wasser",unit="degC"} 10
`,
		},
		{
			name: "op duration empty",
			fn:   c.collectOperatingDuration,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Betriebsstunden",
					},
				},
			},
			want: `
# HELP luxws_operating_duration_seconds Operating time
# TYPE luxws_operating_duration_seconds gauge
luxws_operating_duration_seconds{name=""} 0
`,
		},
		{
			name: "op duration full",
			fn:   c.collectOperatingDuration,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Betriebsstunden",
						Items: luxwsclient.ContentItems{
							{Name: "On\tspace", Value: luxwsclient.String("1h")},
							{Name: "Heat", Value: luxwsclient.String("1:2:3")},
							{Name: "Impulse xyz", Value: luxwsclient.String("")},
						},
					},
				},
			},
			want: `
# HELP luxws_operating_duration_seconds Operating time
# TYPE luxws_operating_duration_seconds gauge
luxws_operating_duration_seconds{name="Heat"} 3723
luxws_operating_duration_seconds{name="On space"} 3600
`,
		},
		{
			name: "op elapsed empty",
			fn:   c.collectElapsedTime,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Ablaufzeiten",
					},
				},
			},
			want: `
# HELP luxws_elapsed_duration_seconds Elapsed time
# TYPE luxws_elapsed_duration_seconds gauge
luxws_elapsed_duration_seconds{name=""} 0
`,
		},
		{
			name: "op elapsed full",
			fn:   c.collectElapsedTime,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Ablaufzeiten",
						Items: luxwsclient.ContentItems{
							{Name: "a", Value: luxwsclient.String("1h")},
							{Name: "b", Value: luxwsclient.String("1:2")},
						},
					},
				},
			},
			want: `
# HELP luxws_elapsed_duration_seconds Elapsed time
# TYPE luxws_elapsed_duration_seconds gauge
luxws_elapsed_duration_seconds{name="a"} 3600
luxws_elapsed_duration_seconds{name="b"} 3720
`,
		},
		{
			name: "inputs empty",
			fn:   c.collectInputs,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Eingänge",
					},
				},
			},
			want: `
# HELP luxws_input Input values
# TYPE luxws_input gauge
luxws_input{name="",unit=""} 0
`,
		},
		{
			name: "inputs full",
			fn:   c.collectInputs,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Eingänge",
						Items: luxwsclient.ContentItems{
							{Name: "temp a", Value: luxwsclient.String("20 °C")},
							{Name: "pressure", Value: luxwsclient.String("3 bar")},
						},
					},
				},
			},
			want: `
# HELP luxws_input Input values
# TYPE luxws_input gauge
luxws_input{name="temp a",unit="degC"} 20
luxws_input{name="pressure",unit="bar"} 3
`,
		},
		{
			name: "outputs empty",
			fn:   c.collectOutputs,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Ausgänge",
					},
				},
			},
			want: `
# HELP luxws_output Output values
# TYPE luxws_output gauge
luxws_output{name="",unit=""} 0
`,
		},
		{
			name: "outputs full",
			fn:   c.collectOutputs,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Ausgänge",
						Items: luxwsclient.ContentItems{
							{Name: "rot", Value: luxwsclient.String("200 RPM")},
							{Name: "pwm", Value: luxwsclient.String("33 %")},
						},
					},
				},
			},
			want: `
# HELP luxws_output Output values
# TYPE luxws_output gauge
luxws_output{name="pwm",unit="pct"} 33
luxws_output{name="rot",unit="rpm"} 200
`,
		},
		{
			name: "supplied heat empty",
			fn:   c.collectSuppliedHeat,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Wärmemenge",
					},
				},
			},
			want: `
# HELP luxws_supplied_heat Supplied heat
# TYPE luxws_supplied_heat gauge
luxws_supplied_heat{name="",unit=""} 0
`,
		},
		{
			name: "supplied heat full",
			fn:   c.collectSuppliedHeat,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Wärmemenge",
						Items: luxwsclient.ContentItems{
							{Name: "water", Value: luxwsclient.String("200 kW")},
							{Name: "ice", Value: luxwsclient.String("100 kW")},
						},
					},
				},
			},
			want: `
# HELP luxws_supplied_heat Supplied heat
# TYPE luxws_supplied_heat gauge
luxws_supplied_heat{name="ice",unit="kW"} 100
luxws_supplied_heat{name="water",unit="kW"} 200
`,
		},
		{
			name: "energy input full",
			fn:   c.collectEnergyInput,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Eingesetzte Energie",
						Items: luxwsclient.ContentItems{
							{Name: "heating", Value: luxwsclient.String("738.2 kWh")},
							{Name: "domestic hot water", Value: luxwsclient.String("238.2 kWh")},
							{Name: "total", Value: luxwsclient.String("976.4 kWh")},
						},
					},
				},
			},
			want: `
# HELP luxws_energy_input Energy Input
# TYPE luxws_energy_input counter
luxws_energy_input{name="heating",unit="kWh"} 738.2
luxws_energy_input{name="domestic hot water",unit="kWh"} 238.2
luxws_energy_input{name="total",unit="kWh"} 976.4
`,
		},
		{
			name: "latest error empty",
			fn:   c.collectLatestError,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Fehlerspeicher",
					},
				},
			},
			want: `
# HELP luxws_latest_error Latest error
# TYPE luxws_latest_error gauge
luxws_latest_error{reason=""} 0
`,
		},
		{
			name: "latest error",
			fn:   c.collectLatestError,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Fehlerspeicher",
						Items: luxwsclient.ContentItems{
							{Name: "02.02.11 08:00:00", Value: luxwsclient.String("aaa")},
							{Name: "03.04.14 23:00:00", Value: luxwsclient.String("bbb")},
							{Name: "01.01.10 09:00:11", Value: luxwsclient.String("aaa")},
						},
					},
				},
			},
			want: `
# HELP luxws_latest_error Latest error
# TYPE luxws_latest_error gauge
luxws_latest_error{reason="aaa"} 1296633600
luxws_latest_error{reason="bbb"} 1396566000
`,
		},
		{
			name: "latest error with empty rows",
			fn:   c.collectLatestError,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Fehlerspeicher",
						Items: luxwsclient.ContentItems{
							{Name: "----", Value: luxwsclient.String("placeholder")},
							{Name: "08.11.21 21:40:09", Value: luxwsclient.String("text")},
							{Name: "----", Value: luxwsclient.String("----")},
						},
					},
				},
			},
			want: `
# HELP luxws_latest_error Latest error
# TYPE luxws_latest_error gauge
luxws_latest_error{reason="text"} 1636407609
`,
		},
		{
			name: "latest switch-off empty",
			fn:   c.collectLatestSwitchOff,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Abschaltungen",
					},
				},
			},
			want: `
# HELP luxws_latest_switchoff Latest switch-off
# TYPE luxws_latest_switchoff gauge
luxws_latest_switchoff{reason=""} 0
`,
		},
		{
			name: "latest switch-off",
			fn:   c.collectLatestSwitchOff,
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{
						Name: "Abschaltungen",
						Items: luxwsclient.ContentItems{
							{Name: "02.02.19 08:00:00", Value: luxwsclient.String("aaa")},
							{Name: "03.04.20 23:00:00", Value: luxwsclient.String("bbb")},
							{Name: "01.01.20 09:00:11", Value: luxwsclient.String("aaa")},
						},
					},
				},
			},
			want: `
# HELP luxws_latest_switchoff Latest switch-off
# TYPE luxws_latest_switchoff gauge
luxws_latest_switchoff{reason="aaa"} 1577869211
luxws_latest_switchoff{reason="bbb"} 1585954800
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &adapter{
				c: c,
				collect: func(ch chan<- prometheus.Metric) error {
					return tc.fn(ch, tc.input, &tc.quirks)
				},
			}
			a.collectAndCompare(t, tc.want, tc.wantErr)

			if diff := cmp.Diff(tc.wantQuirks, tc.quirks, cmp.AllowUnexported(quirks{})); diff != "" {
				t.Errorf("%s failed: Quirks diff (-want +got):\n%s", tc.name, diff)
			}
		})
	}
}

func TestCollectAll(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   *luxwsclient.ContentRoot
		want    string
		wantErr error
	}{
		{
			name:    "empty",
			input:   &luxwsclient.ContentRoot{},
			want:    "",
			wantErr: cmpopts.AnyError,
		},
		{
			name: "complete content",
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{Name: "elapsed times"},
					{Name: "error memory"},
					{Name: "heat quantity"},
					{Name: "energy input"},
					{Name: "information"},
					{Name: "inputs"},
					{Name: "operating hours"},
					{Name: "outputs"},
					{Name: "switch offs"},
					{Name: "system status"},
					{Name: "temperatures"},
				},
			},
			want: `
# HELP luxws_defrost Defrost demand in % and last defrost time
# TYPE luxws_defrost gauge
luxws_defrost{name="demand",unit=""} 0
luxws_defrost{name="last",unit="ts"} -6.21355968e+10
# HELP luxws_elapsed_duration_seconds Elapsed time
# TYPE luxws_elapsed_duration_seconds gauge
luxws_elapsed_duration_seconds{name=""} 0
# HELP luxws_energy_input Energy Input
# TYPE luxws_energy_input counter
luxws_energy_input{name="",unit=""} 0
# HELP luxws_heat_capacity Heat Capacity
# TYPE luxws_heat_capacity gauge
luxws_heat_capacity{unit=""} 0
# HELP luxws_heat_quantity Heat quantity
# TYPE luxws_heat_quantity gauge
luxws_heat_quantity{unit=""} 0
# HELP luxws_impulses Impulses via operating hours
# TYPE luxws_impulses counter
luxws_impulses{name="",unit=""} 0
# HELP luxws_info Controller information
# TYPE luxws_info gauge
luxws_info{hptype="",swversion=""} 1
# HELP luxws_input Input values
# TYPE luxws_input gauge
luxws_input{name="",unit=""} 0
# HELP luxws_latest_error Latest error
# TYPE luxws_latest_error gauge
luxws_latest_error{reason=""} 0
# HELP luxws_latest_switchoff Latest switch-off
# TYPE luxws_latest_switchoff gauge
luxws_latest_switchoff{reason=""} 0
# HELP luxws_operating_duration_seconds Operating time
# TYPE luxws_operating_duration_seconds gauge
luxws_operating_duration_seconds{name=""} 0
# HELP luxws_operational_mode Operational mode
# TYPE luxws_operational_mode gauge
luxws_operational_mode{mode=""} 1
# HELP luxws_output Output values
# TYPE luxws_output gauge
luxws_output{name="",unit=""} 0
# HELP luxws_supplied_heat Supplied heat
# TYPE luxws_supplied_heat gauge
luxws_supplied_heat{name="",unit=""} 0
# HELP luxws_temperature Sensor temperature
# TYPE luxws_temperature gauge
luxws_temperature{name="",unit=""} 0
# HELP luxws_operational_mode_id Operational mode by ID
# TYPE luxws_operational_mode_id gauge
luxws_operational_mode_id{mode=""} 0
`,
		},
		{
			// Heat pump controllers of type L2A don't report the amount of
			// supplied heat.
			//
			// https://github.com/hansmi/wp2reg-luxws/issues/11
			name: "L2A type",
			input: &luxwsclient.ContentRoot{
				Items: luxwsclient.ContentItems{
					{Name: "elapsed times"},
					{Name: "error memory"},
					{Name: "heat quantity"},
					{Name: "energy input"},
					{Name: "information"},
					{Name: "inputs"},
					{Name: "operating hours"},
					{Name: "outputs"},
					{Name: "switch offs"},
					{
						Name: "system status",
						Items: luxwsclient.ContentItems{
							{Name: "type of heat pump", Value: luxwsclient.String("aaa")},
							{Name: "type of heat pump", Value: luxwsclient.String("l2a")},
						},
					},
					{Name: "temperatures"},
				},
			},
			want: `
# HELP luxws_defrost Defrost demand in % and last defrost time
# TYPE luxws_defrost gauge
luxws_defrost{name="demand",unit=""} 0
luxws_defrost{name="last",unit="ts"} -6.21355968e+10
# HELP luxws_elapsed_duration_seconds Elapsed time
# TYPE luxws_elapsed_duration_seconds gauge
luxws_elapsed_duration_seconds{name=""} 0
# HELP luxws_energy_input Energy Input
# TYPE luxws_energy_input counter
luxws_energy_input{name="",unit=""} 0
# HELP luxws_heat_capacity Heat Capacity
# TYPE luxws_heat_capacity gauge
luxws_heat_capacity{unit=""} 0
# HELP luxws_heat_quantity Heat quantity
# TYPE luxws_heat_quantity gauge
luxws_heat_quantity{unit=""} 0
# HELP luxws_impulses Impulses via operating hours
# TYPE luxws_impulses counter
luxws_impulses{name="",unit=""} 0
# HELP luxws_info Controller information
# TYPE luxws_info gauge
luxws_info{hptype="aaa, l2a",swversion=""} 1
# HELP luxws_input Input values
# TYPE luxws_input gauge
luxws_input{name="",unit=""} 0
# HELP luxws_latest_error Latest error
# TYPE luxws_latest_error gauge
luxws_latest_error{reason=""} 0
# HELP luxws_latest_switchoff Latest switch-off
# TYPE luxws_latest_switchoff gauge
luxws_latest_switchoff{reason=""} 0
# HELP luxws_operating_duration_seconds Operating time
# TYPE luxws_operating_duration_seconds gauge
luxws_operating_duration_seconds{name=""} 0
# HELP luxws_operational_mode Operational mode
# TYPE luxws_operational_mode gauge
luxws_operational_mode{mode=""} 1
# HELP luxws_output Output values
# TYPE luxws_output gauge
luxws_output{name="",unit=""} 0
# HELP luxws_temperature Sensor temperature
# TYPE luxws_temperature gauge
luxws_temperature{name="",unit=""} 0
# HELP luxws_operational_mode_id Operational mode by ID
# TYPE luxws_operational_mode_id gauge
luxws_operational_mode_id{mode=""} 0
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCollector(collectorOpts{
				terms: luxwslang.English,
				loc:   time.UTC,
			})

			a := &adapter{
				c: c,
				collect: func(ch chan<- prometheus.Metric) error {
					return c.collectAll(ch, tc.input)
				},
			}
			a.collectAndCompare(t, tc.want, tc.wantErr)
		})
	}
}

func TestCollectHTTP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", "Mon, 02 Jan 2006 15:04:05 GMT")
	}))
	t.Cleanup(server.Close)

	c := newCollector(collectorOpts{
		terms: luxwslang.German,
		loc:   time.UTC,
	})

	if serverURL, err := url.Parse(server.URL); err != nil {
		t.Error(err)
	} else {
		c.httpAddress = serverURL.Host
	}

	want := `
# HELP luxws_node_time_seconds System time in seconds since epoch (1970)
# TYPE luxws_node_time_seconds gauge
luxws_node_time_seconds 1136214245
`

	a := &adapter{
		c: c,
		collect: func(ch chan<- prometheus.Metric) error {
			return c.collectHTTP(ctx, ch)
		},
	}
	a.collectAndCompare(t, want, nil)
}

func TestCollect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	zl, _ := zap.NewDevelopment()
	c := newCollector(collectorOpts{
		terms:   luxwslang.English,
		loc:     time.Local,
		timeout: time.Minute,
		log:     zl,
	})

	if serverURL, err := url.Parse(server.URL); err != nil {
		t.Error(err)
	} else {
		c.address = serverURL.Host
		c.httpAddress = serverURL.Host
	}

	want := `
# HELP luxws_up Whether scrape was successful
# TYPE luxws_up gauge
luxws_up{status="collection via LuxWS protocol failed: websocket: bad handshake"} 0
`

	discardAllLogs(t)

	a := &adapter{
		c: c,
		metricNames: []string{
			"luxws_up",
		},
		collect: func(ch chan<- prometheus.Metric) error {
			c.Collect(ch)
			return nil
		},
	}
	a.collectAndCompare(t, want, nil)
}
