package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var listenAddr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
var beaconChainAddr = flag.String("beacon-chain-address", ":5052", "The address of the beacon chain HTTP API.")
var refreshInterval = flag.Duration("refresh-interval", 5*time.Second, "The interval between polling the beacon-chain for metrics.")

type uint64Flags []uint64

func (i *uint64Flags) String() string {
	return "hi"
}

func (i *uint64Flags) Set(value string) error {
	var intVal uint64
	fmt.Sscan(value, &intVal)

	*i = append(*i, intVal)
	return nil
}

var validatorIndices uint64Flags

func newBalanceGauge(labels map[string]string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   "eth2",
		Subsystem:   "validator",
		Name:        "balance_gwei",
		Help:        "The balance of a given validator.",
		ConstLabels: labels,
	})
}

type balanceMonitor struct {
	ticker         *time.Ticker
	Done           chan bool
	ctx            context.Context
	client         *BeaconChainClient
	balanceGauge   prometheus.Gauge
	validatorIndex uint64
}

func newBalanceMonitor(refreshInterval time.Duration, client *BeaconChainClient, validatorIndex uint64) *balanceMonitor {
	gauge := newBalanceGauge(map[string]string{"validator_index": strconv.FormatUint(validatorIndex, 10)})
	prometheus.MustRegister(gauge)

	monitor := &balanceMonitor{
		ticker:         time.NewTicker(refreshInterval),
		client:         client,
		balanceGauge:   gauge,
		validatorIndex: validatorIndex,
	}

	return monitor
}

type ValidatorInfo struct {
	Balance   uint64 `json:"balance,string"`
	Pubkey    string `json:"pubkey"`
	Validator struct {
		ActivationEligibilityEpoch uint64 `json:"activation_eligibility_epoch,string"`
		ActivationEpoch            uint64 `json:"activation_epoch,string"`
		EffectiveBalance           uint64 `json:"effective_balance,string"`
		ExitEpoch                  uint64 `json:"exit_epoch,string"`
		Pubkey                     string `json:"pubkey"`
		Slashed                    bool   `json:"slashed"`
		WithdrawableEpoch          uint64 `json:"withdrawable_epoch,string"`
		WithdrawalCredentials      string `json:"withdrawal_credentials"`
	} `json:"validator"`
	ValidatorIndex uint64 `json:"validator_index"`
}

type ValidatorResponse struct {
	Data ValidatorInfo `json:"data"`
}

type BeaconChainClient struct {
	baseURL string
	client  *http.Client
}

func NewBeaconChainClient(addr string) (*BeaconChainClient, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:        64,
			MaxIdleConnsPerHost: 64,
			IdleConnTimeout:     384 * time.Second,
		},
	}

	if !strings.HasPrefix(addr, "http") {
		addr = fmt.Sprintf("http://%s", addr)
	}
	_, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	return &BeaconChainClient{
		baseURL: addr,
		client:  client,
	}, nil
}

func (b *BeaconChainClient) GetValidator(validatorIndex uint64) (ValidatorInfo, error) {
	resp, err := b.client.Get(b.baseURL + "/eth/v1/beacon/states/head/validators/" + strconv.FormatUint(validatorIndex, 10))
	if err != nil {
		return ValidatorInfo{}, err
	}

	defer resp.Body.Close()

	r := ValidatorResponse{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return ValidatorInfo{}, err
	}

	return r.Data, nil
}

func (b *balanceMonitor) recordBalance() error {
	validator, err := b.client.GetValidator(b.validatorIndex)
	if err != nil {
		return err
	}
	log.Println(fmt.Sprintf("validator response: %+v", validator))
	b.balanceGauge.Set(float64(validator.Balance))

	return nil
}

func (b *balanceMonitor) Run() {
	log.Println(fmt.Sprintf("Balance monitor started for validatorIndex: %v", b.validatorIndex))
	for {
		select {
		case <-b.Done:
			return
		case <-b.ticker.C:
			err := b.recordBalance()
			if err != nil {
				log.Println(fmt.Sprintf("error recording balances: %+v", err))
			}
		}
	}
}

func main() {
	flag.Var(&validatorIndices, "validator-index", "Validator index to gather metrics on.")
	flag.Parse()

	log.Println(fmt.Sprintf("listen-address: %s", *listenAddr))
	log.Println(fmt.Sprintf("beacon-chain-address: %s", *beaconChainAddr))
	log.Println(fmt.Sprintf("refresh-interval: %v", refreshInterval))

	client, err := NewBeaconChainClient(*beaconChainAddr)
	if err != nil {
		log.Fatalln(err)
	}

	monitors := map[uint64]*balanceMonitor{}
	for _, validatorIndex := range validatorIndices {
		monitor := newBalanceMonitor(*refreshInterval, client, validatorIndex)
		monitors[validatorIndex] = monitor
	}

	for _, monitor := range monitors {
		go monitor.Run()
	}

	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
