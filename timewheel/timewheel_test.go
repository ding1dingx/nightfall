package timewheel

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/shenghui0779/nightfall/worker"
)

// TestTimeWheel 测试时间轮
func TestTimeWheel(t *testing.T) {
	ctx := context.Background()

	ch := make(chan string)
	defer close(ch)

	tw := New(7, time.Second, worker.P())

	addedAt := time.Now()

	tw.Go(ctx, func(ctx context.Context, taskId string, attempts int64) bool {
		ch <- fmt.Sprintf("task[%s][%d] run after %ds", taskId, attempts, int64(math.Round(time.Since(addedAt).Seconds())))
		return attempts < 10
	}, time.Second*time.Duration(1))

	tw.Go(ctx, func(ctx context.Context, taskId string, attempts int64) bool {
		ch <- fmt.Sprintf("task[%s][%d] run after %ds", taskId, attempts, int64(math.Round(time.Since(addedAt).Seconds())))
		return attempts < 5
	}, time.Second*time.Duration(2))

	for i := 0; i < 15; i++ {
		t.Log(<-ch)
	}
}
