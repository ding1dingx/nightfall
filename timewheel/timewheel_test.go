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

	tw.Go(ctx, func(ctx context.Context, taskId string, attempts int64) time.Duration {
		ch <- fmt.Sprintf("task[%s][%d] run after %ds", taskId, attempts, int64(math.Round(time.Since(addedAt).Seconds())))
		if attempts >= 10 {
			return 0
		}
		if attempts%2 == 0 {
			return time.Second * 2
		}
		return time.Second
	}, time.Second)

	tw.Go(ctx, func(ctx context.Context, taskId string, attempts int64) time.Duration {
		ch <- fmt.Sprintf("task[%s][%d] run after %ds", taskId, attempts, int64(math.Round(time.Since(addedAt).Seconds())))
		if attempts >= 5 {
			return 0
		}
		return time.Second * 2
	}, time.Second*2)

	for i := 0; i < 15; i++ {
		t.Log(<-ch)
	}
}
