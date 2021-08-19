[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/fgouteroux/sensu-go-graphite-handler)
![Go Test](https://github.com/fgouteroux/sensu-go-graphite-handler/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/fgouteroux/sensu-go-graphite-handler/workflows/goreleaser/badge.svg)

## Sensu Go Graphite Handler Plugin

- [Overview](#overview)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Sensu Go](#sensu-go)
    - [Asset registration](#asset-registration)
    - [Asset definition](#asset-definition)
    - [Check definition](#check-definition)
    - [Handler definition](#handler-definition)
  - [Sensu Core](#sensu-core)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

### Overview

The Sensu Graphite Handler is a [Sensu Event Handler][3] that sends metrics to the time series database [Graphite][2]. [Sensu][1] can collect metrics using check output metric extraction. Those collected metrics pass through the event pipeline, allowing Sensu to deliver the metrics to the configured metric event handlers. This Graphite handler will allow you to store, instrument, and visualize the metric data from Sensu.

This handler have been copied from [nixwiz][8] and from [kuleuven][9].


Why another graphite handler:

In the original repository from [nixwiz][8], each metric is appended by the entity name and the entity check name.

    myprefix.myhost_domain.com.my-check-name_vmstat.nr_free_pages


As this is not what we expect, we modify this handler to only prefix the metric and take the metric name, like this:

    myprefix.vmstat.nr_free_pages


Main goals:
- Add prefix to the metric name
- Add support to append entity/check labels to the prefix
- Add support to append entity/check annotations to the prefix


## Usage examples

### Help

```
Usage:
  sensu-go-graphite-handler [flags]
  sensu-go-graphite-handler [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -a, --annotations string    The annotations names (comma-separated) of entity/check annotations that must be added to the prefix in graphite for these metrics
  -c, --count                 Count all metrics in event and append to metrics
  -s, --count-scheme string   the string to be prepended to count metric in graphite, could be: my.scheme1 or labels:label1,label2 or annotations:annotation1,annotation2
  -h, --help                  help for sensu-go-graphite-handler
  -H, --host string           the hostname or address of the graphite server (default "127.0.0.1")
  -l, --labels string         The labels names (comma-separated) of entity/check labels that must be added to the prefix in graphite for these metrics
  -n, --no-prefix             Do not include *any* prefixes, use the bare metrics.point.name
  -p, --port uint             the port number to which to connect on the graphite server (default 2003)
  -P, --prefix string         the string to be prepended to each metric in graphite (default "sensu")
      --protocol string       the protocol to which to connect on the graphite server (default "tcp")
```

Labels or Annotations will lookup in event entity, then in event check if not found.

### Example: use entity annotation for metric prefix 

1/ First configure the annotations on your sensu agent.

In /etc/sensu/agent.yaml
```
[...]
annotations:
  short_name: myhost
  short_dc: dc1
```

2/ Create the graphite handler
```json
{
    "api_version": "core/v2",
    "type": "Handler",
    "metadata": {
        "namespace": "default",
        "name": "graphite"
    },
    "spec": {
        "type": "pipe",
        "command": "sensu-go-graphite-handler --prefix myprefix --annotations short_dc -H my-graphite-server.test.com",
        "timeout": 10,
        "filters": [
            "has_metrics"
        ]
    }
}
```

3/ In the check definition
```json
{
    "api_version": "core/v2",
    "type": "CheckConfig",
    "metadata": {
        "namespace": "default",
        "name": "metrics-memory-vmstat"
    },
    "spec": {
        "command": "metrics-memory-vmstat.rb --scheme {{.annotations.short_name}}.vmstat",
        "subscriptions":[
            "dummy"
        ],
        "publish": true,
        "interval": 10,
        "output_metric_format": "graphite_plaintext",
        "output_metric_handlers": [
            "graphite"
        ]
    }
}
```

4/ Metric output:

```
myprefix.dc1.myhost.vmstat.pgalloc_normal 0.000000 1585554685
myprefix.dc1.myhost.vmstat.pgalloc_movable 0.000000 1585554685
myprefix.dc1.myhost.vmstat.pgfree 3971198058.000000 1585554685
[...]
```

## Configuration
### Sensu Go
#### Asset registration

Assets are the best way to make use of this plugin. If you're not using an asset, please consider doing so! If you're using sensuctl 5.13 or later, you can use the following command to add the asset: 

`sensuctl asset add fgouteroux/sensu-go-graphite-handler`

If you're using an earlier version of sensuctl, you can download the asset definition from [this project's Bonsai asset index page][5] or [releases][4] or create an executable script from this source.

From the local path of the sensu-go-graphite-handler repository:
```
go build -o /usr/local/bin/sensu-go-graphite-handler main.go
```

#### Asset definition

```yaml
---
type: Asset
api_version: core/v2
metadata:
  name: sensu-go-graphite-handler
spec:
  url: https://assets.bonsai.sensu.io/793026667633e5cb3e60ba1d063eb5a38ac9cd6b/sensu-go-graphite-handler_0.1.0_linux_amd64.tar.gz
  sha512: af738d13865fdce508fc0c4457ef7473c01639cc92da98590d842eb535db0b51bccdef5c310adf0135b5e3b3677487fe7a1b4370ae3028367bc8117c3fb1824c
```

#### Check definition

```json
{
    "api_version": "core/v2",
    "type": "CheckConfig",
    "metadata": {
        "namespace": "default",
        "name": "metrics-memory-vmstat"
    },
    "spec": {
        "command": "metrics-memory-vmstat.rb --scheme core.{{.annotations.short_name}}.vmstat",
        "subscriptions":[
            "dummy"
        ],
        "publish": true,
        "interval": 10,
        "output_metric_format": "graphite_plaintext",
        "output_metric_handlers": [
            "graphite"
        ]
    }
}
```

#### Handler definition

```json
{
    "api_version": "core/v2",
    "type": "Handler",
    "metadata": {
        "namespace": "default",
        "name": "graphite"
    },
    "spec": {
        "type": "pipe",
        "command": "sensu-go-graphite-handler -H my-graphite-server.test.com",
        "timeout": 10,
        "filters": [
            "has_metrics"
        ]
    }
}
```

That's right, you can collect different types of metrics (ex. Influx, Graphite, OpenTSDB, Nagios, etc.), Sensu will extract and transform them, and this handler will populate them into your Graphite.

### Sensu Core

N/A

## Installation from source

### Sensu Go

See the instructions above for [asset registration][7].

### Sensu Core

Install and setup plugins on [Sensu Core][6].

## Additional notes

N/A

## Contributing

N/A

[1]: https://github.com/sensu/sensu-go
[2]: https://graphiteapp.org
[3]: https://docs.sensu.io/sensu-go/latest/reference/handlers/#how-do-sensu-handlers-work
[4]: https://github.com/fgouteroux/sensu-go-graphite-handler/releases
[5]: https://bonsai.sensu.io/assets/fgouteroux/sensu-go-graphite-handler
[6]: https://docs.sensu.io/sensu-core/latest/installation/installing-plugins/
[7]: #asset-registration
[8]: https://github.com/nixwiz/sensu-go-graphite-handler
[9]: https://github.com/kuleuven/sensu-go-graphite-handler
