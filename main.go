package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	errorCodeLabel = "error_code"
)

var (
	totalCredits = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "openrouter_total_credits",
		},
	)
	totalUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "openrouter_total_usage",
		},
	)
	apiRequestDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name: "openrouter_api_request_duration",
		},
	)
	apiSuccessfulRequests = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "openrouter_api_successful_requests",
		},
	)
	apiFailedRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "openrouter_api_failed_requests",
		},
		[]string{errorCodeLabel},
	)
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	apiEndpoint := getAPIEndpoint()
	token := getAPIToken()
	interval := getInterval()

	metricsEndpoint := getExporterEndpoint()
	metricsAddress := getExporterAddress()
	go func() {
		http.Handle(metricsEndpoint, promhttp.Handler())
		slog.Info("starting metrics server", "address", metricsAddress, "endpoint", metricsEndpoint)
		if err := http.ListenAndServe(metricsAddress, nil); err != nil {
			slog.Error("failed to start metrics server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("starting openrouter watcher watcher", "api_endpoint", apiEndpoint, "interval", interval)

	t := time.NewTicker(interval)
	t.Reset(interval - 1)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-t.C:
			credits, err := getCreditsInfo(ctx, apiEndpoint, token)
			if err != nil {
				slog.Error("failed to get credits info", "error", err)
			}
			totalCredits.Set(credits.Total)
			totalUsage.Set(credits.Usage)
			slog.Info("credits info", "total_credits", credits.Total, "total_usage", credits.Usage)
		}
	}
}

type OpenrouterResponse[T any] struct {
	Data T `json:"data"`

	Error struct {
		Code     int            `json:"code"`
		Message  string         `json:"message"`
		Metadata map[string]any `json:"metadata"`
	} `json:"error"`
}

type Credits struct {
	Total float64 `json:"total_credits"`
	Usage float64 `json:"total_usage"`
}

func getCreditsInfo(ctx context.Context, apiEndpoint string, token string) (_ Credits, err error) {
	var response OpenrouterResponse[Credits]

	startTime := time.Now()
	defer func() {
		apiRequestDuration.Observe(time.Since(startTime).Seconds())
		if err != nil {
			apiFailedRequests.WithLabelValues(strconv.Itoa(response.Error.Code)).Inc()
		} else {
			apiSuccessfulRequests.Inc()
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiEndpoint, nil)
	if err != nil {
		return Credits{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Credits{}, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return Credits{}, err
	}

	if response.Error.Code != 0 {
		return Credits{}, fmt.Errorf("API error (code %d): %s", response.Error.Code, response.Error.Message)
	}

	return response.Data, nil
}

func getAPIEndpoint() string {
	endpoint, ok := os.LookupEnv("OPENROUTER_API_ENDPOINT")
	if !ok {
		return "https://openrouter.ai/api/v1/credits"
	}
	return endpoint
}

func getAPIToken() string {
	token := os.Getenv("OPENROUTER_API_TOKEN")
	if token == "" {
		slog.Error("OPENROUTER_API_TOKEN is not set")
		os.Exit(1)
	}
	return token
}

func getInterval() time.Duration {
	interval, ok := os.LookupEnv("WATCH_INTERVAL")
	if !ok {
		return 15 * time.Second
	}
	duration, err := time.ParseDuration(interval)
	if err != nil {
		slog.Error("WATCH_INTERVAL is not a valid duration", "error", err)
		os.Exit(1)
	}
	return duration
}

func getExporterAddress() string {
	address, ok := os.LookupEnv("EXPORTER_ADDRESS")
	if !ok {
		return ":9080"
	}
	return address
}

func getExporterEndpoint() string {
	endpoint, ok := os.LookupEnv("EXPORTER_ENDPOINT")
	if !ok {
		return "/metrics"
	}
	return endpoint
}
