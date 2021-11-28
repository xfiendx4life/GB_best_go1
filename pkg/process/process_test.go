package process

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"lesson1/pkg/crawler"
	"lesson1/pkg/config"
)

type crawlerStub struct {
	res       chan crawler.CrawlResult
	chngDepth chan int // для изменения глубины поиска
	testFunc  func() []crawler.CrawlResult
}

func (c *crawlerStub) Scan(ctx context.Context, url string, depth int) {
	for _, result := range c.testFunc() {
		c.res <- result
	}
}

// change this when possible
func (c *crawlerStub) ChanResult() <-chan crawler.CrawlResult {
	return c.res

}

// change this when possible
func (c *crawlerStub) ChangeDepth(val int) {
	c.chngDepth <- val
}

func TestProcessResultNoError(t *testing.T) {
	cfg := config.Config{
		MaxDepth:   5,
		MaxResults: 10,
		MaxErrors:  10,
		Url:        "http://127.0.0.1:5500/1.html",
		Timeout:    3,
	}
	c := crawlerStub{
		res: make(chan crawler.CrawlResult),
		testFunc: func() []crawler.CrawlResult {
			return []crawler.CrawlResult {
				{nil, "testTitle1", "testurl"},
				{nil, "testTitle2", "testur2"},
				{nil, "testTitle3", "testurl3"},
			}
		},
	}
	r, w, _ := os.Pipe() // got a couple of connected files
	log.SetOutput(w)
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	go c.Scan(ctx, "url", 1) // send data to chan
	ProcessResult(ctx, cancel, &c, cfg)
	// data is written in ProcessResult
	w.Close()
	data, _ := ioutil.ReadAll(r) // read data
	r.Close()
	for i, item := range strings.Split(string(data), "\n") {
		if i < 3 {
			require.Contains(t, item, fmt.Sprintf("testTitle%d", i+1))
		}
	}
	assert.NotEmpty(t, data)
}

// check if errors are logged
func TestProcessResultError(t *testing.T) {
	cfg := config.Config{
		MaxDepth:   3,
		MaxResults: 10,
		MaxErrors:  10,
		Url:        "http://127.0.0.1:5500/1.html",
		Timeout:    3,
	}
	c := crawlerStub{
		res: make(chan crawler.CrawlResult),
		testFunc: func() []crawler.CrawlResult {
			return []crawler.CrawlResult{
				{fmt.Errorf("testError1"), "testTitle1", "testurl"},
				{fmt.Errorf("testError2"), "testTitle2", "testur2"},
				{fmt.Errorf("testError3"), "testTitle3", "testurl3"},
			}
		},
	}
	r, w, _ := os.Pipe() // got a couple of connected files
	log.SetOutput(w)
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	go c.Scan(ctx, "url", 1) // send data to chan
	ProcessResult(ctx, cancel, &c, cfg)
	// data is written in ProcessResult
	w.Close()
	data, _ := ioutil.ReadAll(r) // read data
	r.Close()
	for i, item := range strings.Split(string(data), "\n") {
		if i < 3 {
			require.Contains(t, item, fmt.Sprintf("testError%d", i+1))
		}
	}
}

func TestProcessResultMaxErrorsCancelFunc(t *testing.T) {
	cfg := config.Config{
		MaxDepth:   10,
		MaxResults: 10,
		MaxErrors:  1, // set 1 max error
		Url:        "http://127.0.0.1:5500/1.html",
		Timeout:    3,
	}
	// we have 3 errors
	c := crawlerStub{
		res: make(chan crawler.CrawlResult),
		testFunc: func() []crawler.CrawlResult {
			return []crawler.CrawlResult{
				{fmt.Errorf("testError1"), "testTitle1", "testurl"},
				{fmt.Errorf("testError2"), "testTitle2", "testur2"},
				{fmt.Errorf("testError3"), "testTitle3", "testurl3"},
			}
		},
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	cancelChecker := false
	cancePointer := &cancelChecker
	go c.Scan(ctx, "url", 1) // send data to chan
	// wrap cancel func
	ProcessResult(ctx, func() {
		cancel()
		*cancePointer = true
	}, &c, cfg)
	require.True(t, cancelChecker)
}

func TestProcessResultMaxResultsCancelFunc(t *testing.T) {
	cfg := config.Config{
		MaxDepth:   10,
		MaxResults: 1, // set 1 max result
		MaxErrors:  1,
		Url:        "http://127.0.0.1:5500/1.html",
		Timeout:    3,
	}
	// we have 3 results
	c := crawlerStub{
		res: make(chan crawler.CrawlResult),
		testFunc: func() []crawler.CrawlResult {
			return []crawler.CrawlResult{
				{nil, "testTitle1", "testurl"},
				{nil, "testTitle2", "testur2"},
				{nil, "testTitle3", "testurl3"},
			}
		},
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	cancelChecker := false
	cancePointer := &cancelChecker
	go c.Scan(ctx, "url", 1) // send data to chan
	// wrap cancel func
	ProcessResult(ctx, func() {
		cancel()
		*cancePointer = true
	}, &c, cfg)
	require.True(t, cancelChecker)
}