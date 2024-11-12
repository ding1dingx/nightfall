package errgroup

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNormal(t *testing.T) {
	var (
		m   = make(map[int]int)
		g   Group
		err error
	)
	for i := 0; i < 4; i++ {
		m[i] = i
	}
	g.Go(func(context.Context) (err error) {
		m[1]++
		return
	})
	g.Go(func(context.Context) (err error) {
		m[2]++
		return
	})
	if err = g.Wait(); err != nil {
		t.Log(err)
	}
	t.Log(m)
}

func sleep1s(context.Context) error {
	time.Sleep(time.Second)
	return nil
}

func TestGOMAXPROCS(t *testing.T) {
	// 没有并发数限制
	g := Group{}
	now := time.Now()
	g.Go(sleep1s)
	g.Go(sleep1s)
	g.Go(sleep1s)
	g.Go(sleep1s)
	err := g.Wait()
	assert.Nil(t, err)
	sec := math.Round(time.Since(now).Seconds())
	if sec != 1 {
		t.FailNow()
	}
	// 限制并发数
	g2 := Group{}
	g2.GOMAXPROCS(2)
	now = time.Now()
	g2.Go(sleep1s)
	g2.Go(sleep1s)
	g2.Go(sleep1s)
	g2.Go(sleep1s)
	err = g2.Wait()
	assert.Nil(t, err)
	sec = math.Round(time.Since(now).Seconds())
	if sec != 2 {
		t.FailNow()
	}
	// context canceled
	g3 := WithContext(context.Background())
	g3.GOMAXPROCS(2)
	g3.Go(func(context.Context) error {
		return errors.New("error for testing errgroup context")
	})
	g3.Go(func(ctx context.Context) error {
		time.Sleep(time.Second)
		select {
		case <-ctx.Done():
			t.Log("caused by", context.Cause(ctx))
		default:
		}
		return nil
	})
	err = g3.Wait()
	assert.NotNil(t, err)
	t.Log(err)
}

func TestRecover(t *testing.T) {
	var (
		g   Group
		err error
	)
	g.Go(func(context.Context) (err error) {
		panic("oh my god!")
	})
	if err = g.Wait(); err != nil {
		t.Log(err)
		return
	}
	t.FailNow()
}

var (
	Web   = fakeSearch("web")
	Image = fakeSearch("image")
	Video = fakeSearch("video")
)

type Result string
type Search func(ctx context.Context, query string) (Result, error)

func fakeSearch(kind string) Search {
	return func(_ context.Context, query string) (Result, error) {
		return Result(fmt.Sprintf("%s result for %q", kind, query)), nil
	}
}

// JustErrors illustrates the use of a Group in place of a sync.WaitGroup to
// simplify goroutine counting and error handling. This example is derived from
// the sync.WaitGroup example at https://golang.org/pkg/sync/#example_WaitGroup.
func ExampleGroup_justErrors() {
	var g Group
	var urls = []string{
		"http://www.golang.org/",
		"http://www.google.com/",
		"http://www.somestupidname.com/",
	}
	for _, _url := range urls {
		// Launch a goroutine to fetch the URL.
		url := _url // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func(context.Context) error {
			// Fetch the URL.
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
			}
			return err
		})
	}
	// Wait for all HTTP fetches to complete.
	if err := g.Wait(); err == nil {
		fmt.Println("Successfully fetched all URLs.")
	}
}

// Parallel illustrates the use of a Group for synchronizing a simple parallel
// task: the "Google Search 2.0" function from
// https://talks.golang.org/2012/concurrency.slide#46, augmented with a Context
// and error-handling.
func ExampleGroup_parallel() {
	Google := func(ctx context.Context, query string) ([]Result, error) {
		g := WithContext(ctx)

		searches := []Search{Web, Image, Video}
		results := make([]Result, len(searches))
		for _i, _search := range searches {
			i, search := _i, _search // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func(context.Context) error {
				result, err := search(ctx, query)
				if err == nil {
					results[i] = result
				}
				return err
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		return results, nil
	}

	results, err := Google(context.Background(), "golang")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, result := range results {
		fmt.Println(result)
	}

	// Output:
	// web result for "golang"
	// image result for "golang"
	// video result for "golang"
}

func TestZeroGroup(t *testing.T) {
	err1 := errors.New("errgroup_test: 1")
	err2 := errors.New("errgroup_test: 2")

	cases := []struct {
		errs []error
	}{
		{errs: []error{}},
		{errs: []error{nil}},
		{errs: []error{err1}},
		{errs: []error{err1, nil}},
		{errs: []error{err1, nil, err2}},
	}

	for _, tc := range cases {
		var g Group

		var firstErr error
		for i, err := range tc.errs {
			err := err
			g.Go(func(context.Context) error { return err })

			if firstErr == nil && err != nil {
				firstErr = err
			}

			if gErr := g.Wait(); gErr != firstErr {
				t.Errorf("after g.Go(func() error { return err }) for err in %v\n"+
					"g.Wait() = %v; want %v", tc.errs[:i+1], err, firstErr)
			}
		}
	}
}

func TestWithCancel(t *testing.T) {
	g := WithContext(context.Background())
	g.Go(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return fmt.Errorf("boom")
	})
	var doneErr error
	g.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			doneErr = ctx.Err()
		}
		return doneErr
	})
	g.Wait()
	if doneErr != context.Canceled {
		t.Error("error should be Canceled")
	}
}
