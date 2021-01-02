# eth2-prometheus-exporter

A prometheus exporter that surfaces metrics from a beaconchain node. Tested with Lighthouse.

## Installation

Install with `go get`:

```sh
go get github.com/landakram/eth2-prometheus-exporter/cmd/eth2-prometheus-exporter
```

Or grab a binary for your platform from the [Releases](https://github.com/landakram/eth2-prometheus-exporter/releases) page.

`eth2-prometheus-exporter` has also been packaged for docker so that it's easy to integrate into a kubernetes cluster. Check out the [repository on docker hub](https://hub.docker.com/r/landakram/eth2-prometheus-exporter).

## Usage

If you are running `eth2-prometheus-exporter` on the same box as your beaconchain node, simply specify the validators that you would like to track by index:

```sh
eth2-prometheus-exporter --validator-indices 12345,98765
```

`eth2-prometheus-exporter` exposes an endpoint, `http://localhost:8080/metrics` by default, that is suitable for scraping by a [prometheus server](https://prometheus.io/).

Several other options are provided:

```sh
eth2-prometheus-exporter -h
Usage of eth2-prometheus-exporter:
  -beacon-chain-address string
    	The address of the beacon chain HTTP API. (default ":5052")
  -listen-address string
    	The address to listen on for HTTP requests. (default ":8080")
  -refresh-interval duration
    	The interval between polling the beacon-chain for metrics. (default 5s)
  -validator-indices string
    	Validator indices on which to gather metrics. Multiple validators may be specified as a comma separated list, e.g. 12345,98765
```

### Available metrics

Only one metric is supported right now, which motivated creation of this tool:

* **`eth2_validator_balance_gwei`**: A gauge that records a given validator's balance in gwei. The metric is labeled by `validator_index`.

## Motivation

Lighthouse does not expose validator balances as part of its metrics server. I wanted to alert based on validator balance as a warning sign that the validator is not correctly performing its duties.
