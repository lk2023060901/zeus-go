package scheduler

import (
	"fmt"

	"github.com/lk2023060901/zeus-go/pkg/logger"
)

func fields(kv ...any) []logger.Field {
	if len(kv) == 0 {
		return nil
	}
	out := make([]logger.Field, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			key = fmt.Sprint(kv[i])
		}
		out = append(out, logger.Field{Key: key, Value: kv[i+1]})
	}
	return out
}
