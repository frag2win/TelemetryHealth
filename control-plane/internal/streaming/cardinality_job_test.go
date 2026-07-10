package streaming

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/axiomhq/hyperloglog"
	"go.uber.org/zap"
)

func TestCardinalityJob_ConcurrentProcess_NoRace(t *testing.T) {
	job := NewCardinalityJob(zap.NewNop())
	ctx := context.Background()

	const numRoutines = 50
	var wg sync.WaitGroup
	wg.Add(numRoutines)

	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer wg.Done()
			sketch := hyperloglog.New14()
			sketch.Insert([]byte("value-1"))
			sketch.Insert([]byte("value-2"))

			err := job.Process(ctx, "tenant-123", "service-A", "user_id", sketch, time.Now())
			if err != nil {
				t.Errorf("Process returned error: %v", err)
			}
		}(i)
	}

	wg.Wait()
}
