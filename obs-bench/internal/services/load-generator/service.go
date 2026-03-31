package load_generator_service

import (
	"context"

	"obs-bench/internal/enum"
)

// IStackLoadGenerator — генерация запросов к query API одного стека.
type IStackLoadGenerator interface {
	GenerateQueries(ctx context.Context, port int) error
}

// ILoadGenerator — фасад для use case: выбор генератора по instrument.
type ILoadGenerator interface {
	GenerateQueries(ctx context.Context, instrument enum.Instrument, port int) error
}
