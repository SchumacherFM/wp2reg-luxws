package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/hansmi/wp2reg-luxws/luxwsclient"
	"github.com/hansmi/wp2reg-luxws/luxwslang"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func findContentItem(r *luxwsclient.ContentRoot, name string) (*luxwsclient.ContentItem, error) {
	found := r.FindByName(name)
	if found == nil {
		return nil, fmt.Errorf("item with name %q not found", name)
	}

	return found, nil
}

type contentCollectFunc func(chan<- prometheus.Metric, *luxwsclient.ContentRoot, *quirks) error

type collector struct {
	log                   *zap.Logger
	httpDo                func(req *http.Request) (*http.Response, error)
	sem                   *semaphore.Weighted
	timeout               time.Duration
	address               string
	password              string
	clientOpts            []luxwsclient.Option
	httpAddress           string
	loc                   *time.Location
	terms                 *luxwslang.Terminology
	upDesc                *prometheus.Desc
	infoDesc              *prometheus.Desc
	temperatureDesc       *prometheus.Desc
	operatingDurationDesc *prometheus.Desc
	elapsedDurationDesc   *prometheus.Desc
	inputDesc             *prometheus.Desc
	outputDesc            *prometheus.Desc
	opModeDesc            *prometheus.Desc
	opModeIDDesc          *prometheus.Desc
	heatQuantityDesc      *prometheus.Desc
	heatCapacityDesc      *prometheus.Desc
	suppliedHeatDesc      *prometheus.Desc
	energyInputDesc       *prometheus.Desc
	latestErrorDesc       *prometheus.Desc
	switchOffDesc         *prometheus.Desc
	nodeTimeDesc          *prometheus.Desc
	impulsesDesc          *prometheus.Desc
	defrostDesc           *prometheus.Desc
}

type collectorOpts struct {
	maxConcurrent int64
	timeout       time.Duration
	address       string
	password      string
	httpAddress   string
	loc           *time.Location
	terms         *luxwslang.Terminology
	log           *zap.Logger
}

func newCollector(opts collectorOpts) *collector {
	clientOpts := []luxwsclient.Option{luxwsclient.WithLogFunc(opts.log)}

	if opts.maxConcurrent < 1 {
		opts.maxConcurrent = 1
	}

	return &collector{
		log:                   opts.log,
		httpDo:                cleanhttp.DefaultClient().Do,
		sem:                   semaphore.NewWeighted(opts.maxConcurrent),
		timeout:               opts.timeout,
		address:               opts.address,
		password:              opts.password,
		clientOpts:            clientOpts,
		httpAddress:           opts.httpAddress,
		loc:                   opts.loc,
		terms:                 opts.terms,
		upDesc:                prometheus.NewDesc("luxws_up", "Whether scrape was successful", []string{"status"}, nil),
		temperatureDesc:       prometheus.NewDesc("luxws_temperature", "Sensor temperature", []string{"name", "unit"}, nil),
		operatingDurationDesc: prometheus.NewDesc("luxws_operating_duration_seconds", "Operating time", []string{"name"}, nil),
		elapsedDurationDesc:   prometheus.NewDesc("luxws_elapsed_duration_seconds", "Elapsed time", []string{"name"}, nil),
		inputDesc:             prometheus.NewDesc("luxws_input", "Input values", []string{"name", "unit"}, nil),
		outputDesc:            prometheus.NewDesc("luxws_output", "Output values", []string{"name", "unit"}, nil),
		infoDesc:              prometheus.NewDesc("luxws_info", "Controller information", []string{"swversion", "hptype"}, nil),
		opModeDesc:            prometheus.NewDesc("luxws_operational_mode", "Operational mode", []string{"mode"}, nil),
		opModeIDDesc:          prometheus.NewDesc("luxws_operational_mode_id", "Operational mode by ID", []string{"mode"}, nil),
		heatQuantityDesc:      prometheus.NewDesc("luxws_heat_quantity", "Heat quantity", []string{"unit"}, nil),
		heatCapacityDesc:      prometheus.NewDesc("luxws_heat_capacity", "Heat Capacity", []string{"unit"}, nil),
		energyInputDesc:       prometheus.NewDesc("luxws_energy_input", "Energy Input", []string{"name", "unit"}, nil),
		suppliedHeatDesc:      prometheus.NewDesc("luxws_supplied_heat", "Supplied heat", []string{"name", "unit"}, nil),
		latestErrorDesc:       prometheus.NewDesc("luxws_latest_error", "Latest error", []string{"reason"}, nil),
		switchOffDesc:         prometheus.NewDesc("luxws_latest_switchoff", "Latest switch-off", []string{"reason"}, nil),
		nodeTimeDesc:          prometheus.NewDesc("luxws_node_time_seconds", "System time in seconds since epoch (1970)", nil, nil),
		impulsesDesc:          prometheus.NewDesc("luxws_impulses", "Impulses via operating hours", []string{"name", "unit"}, nil),
		defrostDesc:           prometheus.NewDesc("luxws_defrost", "Defrost demand in % and last defrost time", []string{"name", "unit"}, nil),
	}
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upDesc
	ch <- c.infoDesc
	ch <- c.temperatureDesc
	ch <- c.operatingDurationDesc
	ch <- c.elapsedDurationDesc
	ch <- c.inputDesc
	ch <- c.outputDesc
	ch <- c.opModeDesc
	ch <- c.opModeIDDesc
	ch <- c.heatQuantityDesc
	ch <- c.heatCapacityDesc
	ch <- c.energyInputDesc
	ch <- c.suppliedHeatDesc
	ch <- c.latestErrorDesc
	ch <- c.switchOffDesc
	ch <- c.nodeTimeDesc
	ch <- c.impulsesDesc
	ch <- c.defrostDesc
}

func (c *collector) parseValue(text string) (float64, string, error) {
	text = strings.TrimSpace(text)

	switch text {
	case c.terms.BoolFalse:
		return 0, "bool", nil

	case c.terms.BoolTrue:
		return 1, "bool", nil
	}

	return c.terms.ParseMeasurement(text)
}

func (c *collector) collectInfo(
	ch chan<- prometheus.Metric,
	content *luxwsclient.ContentRoot,
	q *quirks,
) error {
	var swVersion, opMode, heatOutputUnit, heatCapUnit, defrostDemandUnit string
	var heatOutputValue, heatCapValue, defrostDemandValue float64
	var hpType []string
	var lastDefrost time.Time

	group, err := findContentItem(content, c.terms.NavSystemStatus)
	if err != nil {
		return err
	}

	group.EachNonNil(func(item *luxwsclient.ContentItem) {
		switch item.Name {
		case c.terms.StatusType:
			name := normalizeSpace(*item.Value)

			if strings.EqualFold(name, "L2A") {
				q.missingSuppliedHeat = true
			}

			hpType = append(hpType, name)
		case c.terms.StatusSoftwareVersion:
			swVersion = normalizeSpace(*item.Value)
		case c.terms.StatusOperationMode:
			opMode = normalizeSpace(*item.Value)
			if opMode == "" {
				opMode = "off"
			}
		case c.terms.StatusHeatingCapacity:
			if heatCapValue, heatCapUnit, err = c.parseValue(*item.Value); err != nil {
				c.log.Error("StatusHeatingCapacity parseValue failed", zap.Error(err), zap.Stringp("value", item.Value))
			}
		case c.terms.StatusPowerOutput:
			if heatOutputValue, heatOutputUnit, err = c.parseValue(*item.Value); err != nil {
				c.log.Error("StatusPowerOutput parseValue failed", zap.Error(err), zap.Stringp("value", item.Value))
			}
		case c.terms.StatusDefrostDemand:
			if defrostDemandValue, defrostDemandUnit, err = c.parseValue(*item.Value); err != nil {
				c.log.Error("StatusDefrostDemand parseValue failed", zap.Error(err), zap.Stringp("value", item.Value))
			}
		case c.terms.StatusLastDefrost:
			if lastDefrost, err = c.terms.ParseTimestampShort(*item.Value, c.loc); err != nil {
				c.log.Error("StatusLastDefrost parseValue failed", zap.Error(err), zap.Stringp("value", item.Value))
			}

		}
	})

	sort.Strings(hpType)

	opModeID, ok := c.terms.OperationModeMapping[strings.ToLower(opMode)]
	if !ok && c.log != nil {
		c.log.Error("opMode not configured in code", zap.String("operational_mode", opMode))
		opModeID = -1
	}

	ch <- prometheus.MustNewConstMetric(c.infoDesc, prometheus.GaugeValue, 1, swVersion, strings.Join(hpType, ", "))
	ch <- prometheus.MustNewConstMetric(c.opModeDesc, prometheus.GaugeValue, 1, opMode)
	ch <- prometheus.MustNewConstMetric(c.opModeIDDesc, prometheus.GaugeValue, opModeID, opMode)
	ch <- prometheus.MustNewConstMetric(c.heatQuantityDesc, prometheus.GaugeValue, heatOutputValue, heatOutputUnit)
	ch <- prometheus.MustNewConstMetric(c.heatCapacityDesc, prometheus.GaugeValue, heatCapValue, heatCapUnit)
	ch <- prometheus.MustNewConstMetric(c.defrostDesc, prometheus.GaugeValue, defrostDemandValue, "demand", defrostDemandUnit)
	ch <- prometheus.MustNewConstMetric(c.defrostDesc, prometheus.GaugeValue, float64(lastDefrost.Unix()), "last", "ts")

	return nil
}

func (c *collector) collectMeasurements(
	ch chan<- prometheus.Metric,
	desc *prometheus.Desc,
	content *luxwsclient.ContentRoot,
	groupName string,
	vt prometheus.ValueType, // gauge or counter or ...
	optionalIsAllowed func(s string) bool,
) error {
	group, err := findContentItem(content, groupName)
	if err != nil {
		return err
	}

	var found bool
	group.EachNonNil(func(item *luxwsclient.ContentItem) {
		if optionalIsAllowed != nil && !optionalIsAllowed(item.Name) {
			return
		}

		value, unit, err := c.parseValue(*item.Value)
		if err != nil {
			c.log.Error("parseValue failed", zap.Error(err), zap.Stringp("value", item.Value))
			return
		}

		ch <- prometheus.MustNewConstMetric(desc, vt,
			value, normalizeSpace(item.Name), unit)

		found = true
	})

	if !found {
		ch <- prometheus.MustNewConstMetric(desc, vt,
			0, "", "")
	}

	return nil
}

func (c *collector) collectDurations(
	ch chan<- prometheus.Metric,
	desc *prometheus.Desc,
	content *luxwsclient.ContentRoot,
	groupName string,
	ignoreFn func(string) bool,
) error {
	group, err := findContentItem(content, groupName)
	if err != nil {
		return err
	}

	var found bool

	for _, item := range group.Items {
		if item.Value == nil || (ignoreFn != nil && ignoreFn(item.Name)) {
			continue
		}

		duration, err := c.terms.ParseDuration(*item.Value)
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue,
			duration.Seconds(), normalizeSpace(item.Name))

		found = true
	}

	if !found {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue,
			0, "")
	}

	return nil
}

func (c *collector) collectTimetable(ch chan<- prometheus.Metric, desc *prometheus.Desc, content *luxwsclient.ContentRoot, groupName string) error {
	group, err := findContentItem(content, groupName)
	if err != nil {
		return err
	}

	latest := map[string]time.Time{}

	for _, item := range group.Items {
		tsRaw := normalizeSpace(item.Name)

		if item.Value == nil || strings.Trim(tsRaw, "-") == "" {
			continue
		}

		ts, err := c.terms.ParseTimestamp(tsRaw, c.loc)
		if err != nil {
			return err
		}

		reason := normalizeSpace(*item.Value)

		// Use only the most recent timestamp per reason
		if prev := latest[reason]; prev.IsZero() || prev.Before(ts) {
			latest[reason] = ts
		}
	}

	if len(latest) == 0 {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 0, "")
	} else {
		for reason, ts := range latest {
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(ts.Unix()), reason)
		}
	}

	return nil
}

func (c *collector) collectTemperatures(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, _ *quirks) error {
	return c.collectMeasurements(ch, c.temperatureDesc, content, c.terms.NavTemperatures, prometheus.GaugeValue, nil)
}

func (c *collector) collectOperatingDuration(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, _ *quirks) error {
	return c.collectDurations(ch, c.operatingDurationDesc, content, c.terms.NavOpHours, c.terms.HoursImpulsesFn)
}

func (c *collector) collectElapsedTime(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, _ *quirks) error {
	return c.collectDurations(ch, c.elapsedDurationDesc, content, c.terms.NavElapsedTimes, nil)
}

func (c *collector) collectInputs(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, _ *quirks) error {
	return c.collectMeasurements(ch, c.inputDesc, content, c.terms.NavInputs, prometheus.GaugeValue, nil)
}

func (c *collector) collectOutputs(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, _ *quirks) error {
	return c.collectMeasurements(ch, c.outputDesc, content, c.terms.NavOutputs, prometheus.GaugeValue, nil)
}

func (c *collector) collectImpulses(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, _ *quirks) error {
	return c.collectMeasurements(ch, c.impulsesDesc, content, c.terms.NavOpHours, prometheus.CounterValue, c.terms.HoursImpulsesFn)
}

func (c *collector) collectSuppliedHeat(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, q *quirks) error {
	if q.missingSuppliedHeat {
		return nil
	}
	// not a counter because during defrost the heat amount goes down
	return c.collectMeasurements(ch, c.suppliedHeatDesc, content, c.terms.NavHeatQuantity, prometheus.GaugeValue, nil)
}

func (c *collector) collectEnergyInput(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, _ *quirks) error {
	return c.collectMeasurements(ch, c.energyInputDesc, content, c.terms.NavEnergyInput, prometheus.CounterValue, nil)
}

func (c *collector) collectLatestError(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, _ *quirks) error {
	return c.collectTimetable(ch, c.latestErrorDesc, content, c.terms.NavErrorMemory)
}

func (c *collector) collectLatestSwitchOff(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot, _ *quirks) error {
	return c.collectTimetable(ch, c.switchOffDesc, content, c.terms.NavSwitchOffs)
}

func (c *collector) collectAll(ch chan<- prometheus.Metric, content *luxwsclient.ContentRoot) error {
	var err error
	var q quirks

	for _, fn := range []contentCollectFunc{
		c.collectInfo,
		c.collectTemperatures,
		c.collectOperatingDuration,
		c.collectElapsedTime,
		c.collectInputs,
		c.collectOutputs,
		c.collectSuppliedHeat,
		c.collectEnergyInput,
		c.collectLatestError,
		c.collectLatestSwitchOff,
		c.collectImpulses,
	} {
		multierr.AppendInto(&err, fn(ch, content, &q))
	}

	return err
}

func (c *collector) collectWebSocket(ctx context.Context, ch chan<- prometheus.Metric) error {
	cl, err := luxwsclient.Dial(ctx, c.address, c.clientOpts...)
	if err != nil {
		return err
	}

	defer cl.Close()

	nav, err := cl.Login(ctx, c.password)
	if err != nil {
		return err
	}

	info := nav.FindByName(c.terms.NavInformation)
	if info == nil {
		return errors.New("information ID not found in response")
	}

	content, err := cl.Get(ctx, info.ID)
	if err != nil {
		return fmt.Errorf("fetching ID %q failed: %w", info.ID, err)
	}

	return c.collectAll(ch, content)
}

func (c *collector) collectHTTP(ctx context.Context, ch chan<- prometheus.Metric) error {
	url := url.URL{
		Scheme: "http",
		Host:   c.httpAddress,
		Path:   "/",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpDo(req)
	if err != nil {
		return err
	}

	if dateHeader := resp.Header.Get("Date"); dateHeader != "" {
		ts, err := http.ParseTime(dateHeader)
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(c.nodeTimeDesc, prometheus.GaugeValue,
			float64(ts.Unix()))
		return nil
	}
	return errors.New("HTTP header missing server time")
}

func (c *collector) collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	// Limit concurrent collections
	if err := c.sem.Acquire(ctx, 1); err != nil {
		return err
	}

	defer c.sem.Release(1)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := c.collectWebSocket(ctx, ch); err != nil {
			return fmt.Errorf("collection via LuxWS protocol failed: %w", err)
		}

		return nil
	})

	if c.httpAddress != "" {
		g.Go(func() error {
			if err := c.collectHTTP(ctx, ch); err != nil {
				return fmt.Errorf("collection via HTTP protocol failed: %w", err)
			}

			return nil
		})
	}

	return g.Wait()
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if err := c.collect(ctx, ch); err == nil {
		ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 1, "")
	} else {
		c.log.Error("Scrape failed", zap.Error(err))
		ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 0, err.Error())
	}
}
