package influx

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"go.uber.org/zap"
	"telemetry-system/internal/config"
)

type InfluxClient struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
	queryAPI api.QueryAPI
	logger   *zap.Logger
	bucket   string
	org      string
}

func NewInfluxClient(cfg *config.InfluxDBConfig, logger *zap.Logger) (*InfluxClient, error) {
	client := influxdb2.NewClient(cfg.URL, cfg.Token)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Health(ctx); err != nil {
		return nil, fmt.Errorf("InfluxDB health check failed: %w", err)
	}

	writeAPI := client.WriteAPI(cfg.Org, cfg.Bucket)
	queryAPI := client.QueryAPI(cfg.Org)

	return &InfluxClient{
		client:   client,
		writeAPI: writeAPI,
		queryAPI: queryAPI,
		logger:   logger,
		bucket:   cfg.Bucket,
		org:      cfg.Org,
	}, nil
}

func (i *InfluxClient) WriteAPI() api.WriteAPI {
	return i.writeAPI
}

func (i *InfluxClient) QueryAPI() api.QueryAPI {
	return i.queryAPI
}

func (i *InfluxClient) Flush() {
	i.writeAPI.Flush()
}

func (i *InfluxClient) Close() {
	i.client.Close()
	i.logger.Info("InfluxDB client closed")
}

func (i *InfluxClient) Health(ctx context.Context) error {
	_, err := i.client.Health(ctx)
	return err
}