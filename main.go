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
	Count       bool
	CountScheme string
	Port        uint64
	Host        string
	Protocol    string
}

const (
	// flags
	prefix      = "prefix"
	labels      = "labels"
	annotations = "annotations"
	noPrefix    = "no-prefix"
	count       = "count"
	countScheme = "count-scheme"
	port        = "port"
	host        = "host"
	protocol    = "protocol"

	// defaults
	defaultPrefix   = "sensu"
	defaultPort     = 2003
	defaultHost     = "127.0.0.1"
	defaultProtocol = "tcp"
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
			Usage:     "The labels names (comma-separated) of entity/check labels that must be added to the prefix in graphite for these metrics",
			Value:     &config.Labels,
		},
		{
			Path:      annotations,
			Argument:  annotations,
			Shorthand: "a",
			Default:   "",
			Usage:     "The annotations names (comma-separated) of entity/check annotations that must be added to the prefix in graphite for these metrics",
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
			Path:      count,
			Argument:  count,
			Shorthand: "c",
			Default:   false,
			Usage:     "Count all metrics in event and append to metrics",
			Value:     &config.Count,
		},
		{
			Path:      countScheme,
			Argument:  countScheme,
			Shorthand: "s",
			Default:   "",
			Usage:     "the string to be prepended to count metric in graphite, could be: my.scheme1 or labels:label1,label2 or annotations:annotation1,annotation2",
			Value:     &config.CountScheme,
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
		{
			Path:     protocol,
			Argument: protocol,
			Default:  defaultProtocol,
			Usage:    "the protocol to which to connect on the graphite server",
			Value:    &config.Protocol,
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
		config.Prefix = ""
		config.Labels = ""
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
		tmp_name       string
	)

	prefix := config.Prefix
	sanitizedChars := strings.NewReplacer("/", "_", "@", "_", " ", "_")

	for _, label := range strings.Split(config.Labels, ",") {
		if val, ok := event.Entity.Labels[label]; ok {
			prefix = fmt.Sprintf("%s.%s", prefix, sanitizedChars.Replace(val))
		} else if val, ok := event.Check.Labels[label]; ok {
			prefix = fmt.Sprintf("%s.%s", prefix, sanitizedChars.Replace(val))
		}
	}

	for _, annotation := range strings.Split(config.Annotations, ",") {
		if val, ok := event.Entity.Annotations[annotation]; ok {
			prefix = fmt.Sprintf("%s.%s", prefix, sanitizedChars.Replace(val))
		} else if val, ok := event.Check.Annotations[annotation]; ok {
			prefix = fmt.Sprintf("%s.%s", prefix, sanitizedChars.Replace(val))
		}
	}

	var err error
	var g *graphite.Graphite
	if config.Protocol == "udp" {
		g, err = graphite.NewGraphiteUDP(config.Host, int(config.Port))
	} else {
		g, err = graphite.NewGraphite(config.Host, int(config.Port))
	}
	if err != nil {
		return err
	}

	for _, point := range event.Metrics.Points {
		if config.NoPrefix {
			tmpvalue := fmt.Sprintf("%f", point.Value)
			tmp_point_name = sanitizedChars.Replace(point.Name)
			metrics = append(metrics, graphite.NewMetric(tmp_point_name, tmpvalue, point.Timestamp))
		} else {
			// Deal with special cases, such as disk checks that return file system paths as the name
			// Graphite places these on disk using the name, so using any slashes would cause confusion and lost metrics
			if point.Name == "/" {
				tmp_point_name = "root"
			} else {
				tmp_point_name = sanitizedChars.Replace(point.Name)
			}
			tmpname := fmt.Sprintf("%s.%s", prefix, tmp_point_name)
			tmpvalue := fmt.Sprintf("%f", point.Value)
			metrics = append(metrics, graphite.NewMetric(tmpname, tmpvalue, point.Timestamp))
		}
	}

	if config.Count {

		if strings.Contains(config.CountScheme, "labels:") && strings.Contains(config.CountScheme, "annotations:") {
			return fmt.Errorf("usage of labels and annotations are mutually exclusive in count-scheme")
		}
		prefix_count := config.Prefix
		if config.CountScheme != "" {

			if strings.Contains(config.CountScheme, "labels:") {
				labels := strings.Split(config.CountScheme, "labels:")

				for _, label := range strings.Split(labels[1], ",") {
					if val, ok := event.Entity.Labels[label]; ok {
						prefix_count = fmt.Sprintf("%s.%s", prefix_count, sanitizedChars.Replace(val))
					} else if val, ok := event.Check.Labels[label]; ok {
						prefix_count = fmt.Sprintf("%s.%s", prefix_count, sanitizedChars.Replace(val))
					}
				}
			}

			if strings.Contains(config.CountScheme, "annotations:") {

				annotations := strings.Split(config.CountScheme, "annotations:")
				for _, annotation := range strings.Split(annotations[1], ",") {
					if val, ok := event.Entity.Annotations[annotation]; ok {
						prefix_count = fmt.Sprintf("%s.%s", prefix_count, sanitizedChars.Replace(val))
					} else if val, ok := event.Check.Annotations[annotation]; ok {
						prefix_count = fmt.Sprintf("%s.%s", prefix_count, sanitizedChars.Replace(val))
					}
				}
			}
		}

		metric_count := fmt.Sprintf("%d", len(event.Metrics.Points))

		if config.NoPrefix {
			tmp_name = fmt.Sprintf("%s.%s", event.Check.Name, "count")
		} else {
			tmp_name = fmt.Sprintf("%s.%s.%s", prefix_count, event.Check.Name, "count")
		}
		metrics = append(metrics, graphite.NewMetric(tmp_name, metric_count, event.Timestamp))
	}

	err = g.SendMetrics(metrics)
	errClose := g.Disconnect()
	if errClose != nil {
		err = fmt.Errorf("SendMetrics error: %v. Disconnect error: %v", err, errClose)
	}

	return err
}
