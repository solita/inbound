package metrics

type Collector interface {
	// Add one error to metrics
	ReceiveError()
	// Add one success to metrics, and if supported, receive time to another metric
	ReceiveSuccess(timeMs int64)
}
