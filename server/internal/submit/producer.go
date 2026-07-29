package submit

import "context"

type ProducerInterface interface {
	Send(ctx context.Context, key string, value any) error
}
