package main

import (
	"fmt"
	"strings"

	"github.com/marpaia/graphite-golang"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
)

type HandlerConfig struct {
	sensu.PluginConfig
	Prefix      string
	Labels      string
	Annotations string
	NoPrefix    bool
	Port        uint64
	Host        string
}

const (
	// flags
	prefix      = "prefix"
	labels      = "labels"
	annotations = "annotations"
	noPrefix    = "no-prefix"
	port        = "port"
	host        = "host"

	// defaults
	defaultPrefix = "sensu"
	defaultPort   = 2003
	defaultHost   = "127.0.0.1"
)

var (
	metricPrefix string

	config = HandlerConfig{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-go-graphite-handler",
			Short:    "The Sensu Go Graphite for sending metrics to Carbon/Graphite",
			Keyspace: "sensu.io/plugins/graphite/config",
		},
	}

	graphiteConfigOptions = []*sensu.PluginConfigOption{
		{
			Path:      prefix,
			Argument:  prefix,
			Shorthand: "P",
			Default:   defaultPrefix,
			Usage:     "the string to be prepended to each metric in graphite",
			Value:     &config.Prefix,
		},
		{
			Path:      labels,
			Argument:  labels,
			Shorthand: "l",
			Default:   "",
			Usage:     "The labels names (comma-separated) of entity labels that must be added to the prefix in graphite for these metrics",
			Value:     &config.Labels,
		},
		{
			Path:      annotations,
			Argument:  annotations,
			Shorthand: "a",
			Default:   "",
			Usage:     "The annotations names (comma-separated) of entity annotations that must be added to the prefix in graphite for these metrics",
			Value:     &config.Annotations,
		},
		{
			Path:      noPrefix,
			Argument:  noPrefix,
			Shorthand: "n",
			Default:   false,
			Usage:     "Do not include *any* prefixes, use the bare metrics.point.name",
			Value:     &config.NoPrefix,
		},
		{
			Path:      port,
			Argument:  port,
			Shorthand: "p",
			Default:   uint64(defaultPort),
			Usage:     "the port number to which to connect on the graphite server",
			Value:     &config.Port,
		},
		{
			Path:      host,
			Argument:  host,
			Shorthand: "H",
			Default:   defaultHost,
			Usage:     "the hostname or address of the graphite server",
			Value:     &config.Host,
		},
	}
)

func main() {
	goHandler := sensu.NewGoHandler(&config.PluginConfig, graphiteConfigOptions, CheckArgs, SendMetrics)
	goHandler.Execute()
}

func CheckArgs(event *corev2.Event) error {
	if !event.HasMetrics() {
		return fmt.Errorf("event does not contain metrics")
	}

	if config.NoPrefix {
		config.Prefix      = ""
		config.Labels      = ""
		config.Annotations = ""
	}

	if config.Labels != "" && config.Annotations != "" {
		return fmt.Errorf("usage of labels and annotations are mutually exclusive")
	}

	return nil
}

func SendMetrics(event *corev2.Event) error {
	var (
		metrics        []graphite.Metric
		tmp_point_name string
	)

	prefix := config.Prefix

	for _, label := range strings.Split(config.Labels, ",") {
		if val, ok := event.Entity.Labels[label]; ok {
			prefix = fmt.Sprintf("%s.%s", prefix, val)
		}
	}

	for _, annotation := range strings.Split(config.Annotations, ",") {
		if val, ok := event.Entity.Annotations[annotation]; ok {
			prefix = fmt.Sprintf("%s.%s", prefix, val)
		}
	}

	g, err := graphite.NewGraphite(config.Host, int(config.Port))
	if err != nil {
		return err
	}

	for _, point := range event.Metrics.Points {
		if config.NoPrefix {
			tmpvalue := fmt.Sprintf("%f", point.Value)
			metrics = append(metrics, graphite.NewMetric(point.Name, tmpvalue, point.Timestamp))
			//log.Println("%s %d %d", point.Name, tmpvalue, point.Timestamp)
		} else {
			// Deal with special cases, such as disk checks that return file system paths as the name
			// Graphite places these on disk using the name, so using any slashes would cause confusion and lost metrics
			if point.Name == "/" {
				tmp_point_name = "root"
			} else {
				tmp_point_name = strings.Replace(point.Name, "/", "_", -1)
			}
			tmpname := fmt.Sprintf("%s.%s", prefix, tmp_point_name)
			tmpvalue := fmt.Sprintf("%f", point.Value)
			metrics = append(metrics, graphite.NewMetric(tmpname, tmpvalue, point.Timestamp))
		}
	}

	if err = g.SendMetrics(metrics); err != nil {
		return err
	}

	return g.Disconnect()
}
