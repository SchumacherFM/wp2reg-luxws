package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/alecthomas/kingpin/v2"
	"github.com/hansmi/wp2reg-luxws/luxwslang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	promslogflag "github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"go.uber.org/zap"
)

var (
	webConfig              = webflag.AddFlags(kingpin.CommandLine, ":8081")
	metricsPath            = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	disableExporterMetrics = kingpin.Flag("web.disable-exporter-metrics", "Exclude metrics about the exporter itself").Bool()
	maxConcurrent          = kingpin.Flag("web.max-requests", "Maximum number of concurrent scrape requests").Default("3").Uint()
)

var (
	verbose = kingpin.Flag("verbose", "Log sent and received messages").Bool()
	timeout = kingpin.Flag("scrape-timeout", "Maximum duration for a scrape").Default("1m").Duration()
)

var (
	target = kingpin.Flag("controller.address",
		`host:port for controller Websocket service (e.g. "192.0.2.1:8214")`).PlaceHolder("HOST:PORT").Required().String()
	password = kingpin.Flag("controller.password",
		`password for controller Websocket service`).String()
	httpTarget = kingpin.Flag("controller.address.http",
		`host:port for controller HTTP service; used to retrieve time (e.g. "192.0.2.1:80")`).PlaceHolder("HOST:PORT").String()
)

var timezone = kingpin.Flag("controller.timezone",
	"Timezone for parsing timestamps").Default(time.Local.String()).String()

var lang = kingpin.Flag("controller.language",
	fmt.Sprintf("Controller interface language (one of %q)", supportedLanguages())).PlaceHolder("NAME").Required().String()

func supportedLanguages() []string {
	result := []string{}

	for _, terms := range luxwslang.All() {
		result = append(result, terms.ID)
	}

	return result
}

func main() {
	promslogConfig := &promslog.Config{}
	promslogflag.AddFlags(kingpin.CommandLine, promslogConfig)

	kingpin.Parse()

	//var zapOpts []zap.Option
	//if *verbose {
	//	zapOpts = append(zapOpts,
	//		zap.IncreaseLevel(zap.DebugLevel),
	//		zap.AddStacktrace(zap.DebugLevel),
	//		zap.AddCaller(),
	//	)
	//}
	//zapOpts = append(zapOpts)

	zapencCfg := zap.NewProductionEncoderConfig()
	zapencCfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder

	zapLvl := zap.InfoLevel
	if *verbose {
		zapLvl = zap.DebugLevel
	}
	zaplog := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zapencCfg),
		zapcore.AddSync(os.Stdout),
		zapLvl,
	))
	// zaplog.WithOptions(zapOpts...)

	defer zaplog.Sync()
	opts := collectorOpts{
		maxConcurrent: int64(*maxConcurrent),
		timeout:       *timeout,
		address:       *target,
		password:      *password,
		httpAddress:   *httpTarget,
		log:           zaplog,
	}

	if loc, err := time.LoadLocation(*timezone); err != nil {
		zaplog.Fatal("Loading timezone", zap.Error(err), zap.Stringp("zone", timezone))
	} else {
		opts.loc = loc
	}

	if terms, err := luxwslang.LookupByID(*lang); err != nil {
		zaplog.Fatal("Unknown controller language", zap.Error(err))
	} else {
		opts.terms = terms
	}

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(newCollector(opts))
	if !*disableExporterMetrics {
		reg.MustRegister(
			collectors.NewBuildInfoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
			collectors.NewGoCollector(),
			version.NewCollector("luxws_exporter"),
		)
	}

	http.Handle(*metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>LuxWS Exporter</title></head>
			<body>
			<h1>LuxWS Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	server := &http.Server{}

	if err := web.ListenAndServe(server, webConfig, wraplog{zaplog}); err != nil {
		zaplog.Fatal("ListenAndServe failed", zap.Error(err))
	}
}

type wraplog struct {
	*zap.Logger
}

func (w wraplog) Log(keyvals ...interface{}) error {
	keylen := len(keyvals)

	var level string
	var msg string
	data := make([]zap.Field, 0, (keylen/2)+1)
	for i := 0; i < keylen; i += 2 {
		key := fmt.Sprint(keyvals[i])
		switch key {
		case "level":
			level = keyvals[i+1].(fmt.Stringer).String()
		case "msg":
			msg = keyvals[i+1].(string)
		default:
			data = append(data, zap.Any(key, keyvals[i+1]))
		}
	}

	switch level {
	case "debug":
		w.Debug(msg, data...)
	case "info":
		w.Info(msg, data...)
	case "warn":
		w.Warn(msg, data...)
	case "error":
		w.Error(msg, data...)
	case "fatal":
		w.Fatal(msg, data...)
	}
	return nil
}
