package converters

import "errors"

// Ошибки валидации protobuf метрик
var (
	ErrMissingID    = errors.New("metric ID is required")
	ErrMissingValue = errors.New("value is required for gauge metric")
	ErrMissingDelta = errors.New("delta is required for counter metric")
	ErrUnknownType  = errors.New("unknown metric type")
)
