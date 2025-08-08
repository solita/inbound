package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchCollector struct {
	cw         *cloudwatch.Client
	namespace  string
	dimensions []types.Dimension
	timeout    time.Duration
}

func NewCloudWatchCollector(cw *cloudwatch.Client, namespace string, dimensions map[string]string) *CloudWatchCollector {
	dims := make([]types.Dimension, 0, len(dimensions))
	for k, v := range dimensions {
		k, v := k, v
		dims = append(dims, types.Dimension{Name: &k, Value: &v})
	}
	return &CloudWatchCollector{
		cw:         cw,
		namespace:  namespace,
		dimensions: dims,
		timeout:    10 * time.Second, // per-call timeout
	}
}

// Add one error to metrics
func (c *CloudWatchCollector) ReceiveError() {
	now := time.Now()
	one := 1.0
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	_, err := c.cw.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: &c.namespace,
		MetricData: []types.MetricDatum{
			{
				MetricName: strPtr("ReceiveError"),
				Timestamp:  &now,
				Dimensions: c.dimensions,
				Unit:       types.StandardUnitCount,
				Value:      &one,
			},
		},
	})
	if err != nil {
		slog.Error("Failed to send CloudWatch metric for ReceiveError", "error", err)
	}
}

// Add one success and a latency metric (milliseconds)
func (c *CloudWatchCollector) ReceiveSuccess(timeMs int64) {
	now := time.Now()
	one := 1.0
	lat := float64(timeMs)
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	_, err := c.cw.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: &c.namespace,
		MetricData: []types.MetricDatum{
			{
				MetricName: strPtr("ReceiveSuccess"),
				Timestamp:  &now,
				Dimensions: c.dimensions,
				Unit:       types.StandardUnitCount,
				Value:      &one,
			},
			{
				MetricName: strPtr("ReceiveLatency"),
				Timestamp:  &now,
				Dimensions: c.dimensions,
				Unit:       types.StandardUnitMilliseconds,
				Value:      &lat,
			},
		},
	})
	if err != nil {
		slog.Error("Failed to send CloudWatch metric for ReceiveSuccess", "error", err)
	}
}

func strPtr(s string) *string { return &s }
