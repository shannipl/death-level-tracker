package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TrackedDeaths = promauto.NewCounter(prometheus.CounterOpts{
		Name: "death_tracker_deaths_total",
		Help: "The total number of tracked deaths",
	})

	TrackedLevelUps = promauto.NewCounter(prometheus.CounterOpts{
		Name: "death_tracker_level_ups_total",
		Help: "The total number of tracked level ups",
	})

	TibiaDataRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tibiadata_request_duration_seconds",
		Help:    "Duration of TibiaData API requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"endpoint", "status"})

	TibiaDataRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tibiadata_requests_total",
		Help: "Total number of TibiaData API requests",
	}, []string{"endpoint", "status"})

	TibiaComRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tibiacom_request_duration_seconds",
		Help:    "Duration of Tibia.com HTML scraping requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"status"})

	TibiaComRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tibiacom_requests_total",
		Help: "Total number of Tibia.com HTML scraping requests",
	}, []string{"status"})

	DiscordMessagesSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "discord_messages_sent_total",
		Help: "Total number of Discord messages sent",
	}, []string{"channel_type", "status"})
)
