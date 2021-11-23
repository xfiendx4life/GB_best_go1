package main

import (
	"context"
	"fmt"
	"strings"

	// "fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type crawlerStub struct {
	r         requesterMock
	res       chan CrawlResult
	chngDepth chan int // для изменения глубины поиска
	visited   map[string]struct{}
	mu        sync.RWMutex
	testFunc  func() []CrawlResult
}

type requesterMock struct {
	requestCounter int // to cunt number of requests
	returnsError   bool
}

func (r *requesterMock) Get(ctx context.Context, url string) (Page, error) {
	r.requestCounter += 1
	if r.returnsError {
		r.returnsError = false
		return nil, fmt.Errorf("test error")
	}
	ps := pageStub{
		requestCounter: r.requestCounter,
	}
	return &ps, nil
}

type pageStub struct {
	requestCounter int // sent from requester
}

func (p *pageStub) GetTitle(ctx context.Context) string {
	// as crawler uses interface that's the way to send counter to test function
	return fmt.Sprintf("stubPageTitle%d", p.requestCounter)
}

func (p *pageStub) GetLinks(ctx context.Context) []string {

	return []string{
		"firstlink",
		"secondlink",
		"thirdlink",
		"fourthlink",
		"fifthlink",
		"sixthlink",
	}[:p.requestCounter]
}

// как тут изменить??? чтобы вариации сделать??

func (c *crawlerStub) Scan(ctx context.Context, url string, depth int) {
	for _, result := range c.testFunc() {
		c.res <- result
	}
}

// change this when possible
func (c *crawlerStub) ChanResult() <-chan CrawlResult {
	return c.res

}

// change this when possible
func (c *crawlerStub) ChangeDepth(val int) {
	c.chngDepth <- val
}

// test if the results are logged
func TestProcessResultNoError(t *testing.T) {
	cfg := Config{
		MaxDepth:   5,
		MaxResults: 10,
		MaxErrors:  10,
		Url:        "http://127.0.0.1:5500/1.html",
		Timeout:    3,
	}
	c := crawlerStub{
		res: make(chan CrawlResult),
		testFunc: func() []CrawlResult {
			return []CrawlResult{
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
	processResult(ctx, cancel, &c, cfg)
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
	cfg := Config{
		MaxDepth:   3,
		MaxResults: 10,
		MaxErrors:  10,
		Url:        "http://127.0.0.1:5500/1.html",
		Timeout:    3,
	}
	c := crawlerStub{
		res: make(chan CrawlResult),
		testFunc: func() []CrawlResult {
			return []CrawlResult{
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
	processResult(ctx, cancel, &c, cfg)
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
	cfg := Config{
		MaxDepth:   10,
		MaxResults: 10,
		MaxErrors:  1, // set 1 max error
		Url:        "http://127.0.0.1:5500/1.html",
		Timeout:    3,
	}
	// we have 3 errors
	c := crawlerStub{
		res: make(chan CrawlResult),
		testFunc: func() []CrawlResult {
			return []CrawlResult{
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
	processResult(ctx, func() {
		cancel()
		*cancePointer = true
	}, &c, cfg)
	require.True(t, cancelChecker)
}

func TestProcessResultMaxResultsCancelFunc(t *testing.T) {
	cfg := Config{
		MaxDepth:   10,
		MaxResults: 1, // set 1 max result
		MaxErrors:  1,
		Url:        "http://127.0.0.1:5500/1.html",
		Timeout:    3,
	}
	// we have 3 results
	c := crawlerStub{
		res: make(chan CrawlResult),
		testFunc: func() []CrawlResult {
			return []CrawlResult{
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
	processResult(ctx, func() {
		cancel()
		*cancePointer = true
	}, &c, cfg)
	require.True(t, cancelChecker)
}

func TestScanDepth1(t *testing.T) {
	cfg := Config{
		MaxDepth:   1,
		MaxResults: 1, // set 1 max result
		MaxErrors:  1,
		Url:        "http://testUrl/1.html",
		Timeout:    100,
	}
	r := requesterMock{
		requestCounter: 0,
	}
	cr := NewCrawler(&r)
	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, time.Second*1)
	go cr.Scan(ctx, cfg.Url, cfg.MaxDepth)
	select {
	case <-ctx.Done():
		require.True(t, false)
		return
	case msg := <-cr.ChanResult():
		assert.Nil(t, msg.Err)
		assert.Equal(t, msg.Title, "stubPageTitle1")
	}
}

func TestScanCheckDepth(t *testing.T) {
	cfg := Config{
		MaxDepth: 3,
		// set 3 max result equals to depth,
		// either for loop waits for more
		MaxResults: 3,
		MaxErrors:  1,
		Url:        "http://testUrl/1.html",
		Timeout:    100,
	}
	r := requesterMock{
		requestCounter: 0,
	}
	cr := NewCrawler(&r)
	ctx := context.Background()
	// ctx, _ = context.WithTimeout(ctx, time.Second*1)
	go cr.Scan(ctx, cfg.Url, cfg.MaxDepth)
	res := ""
	_ = res
	var err error
	func() {
		for {
			select {
			case <-ctx.Done():
				require.True(t, false)
				return
			case msg := <-cr.ChanResult():
				if cfg.MaxResults > 0 {
					cfg.MaxResults--
					res = msg.Title
					err = msg.Err
				} else {
					return
				}
			}
		}
	}()

	assert.Nil(t, err)
	assert.NotEqual(t, res, "")
	assert.Equal(t, res, "stubPageTitle3")
}

func TestScanError(t *testing.T) {
	cfg := Config{
		MaxDepth: 3,
		// set 3 max result equals to depth,
		// either for loop waits for more
		MaxResults: 3,
		MaxErrors:  3,
		Url:        "http://testUrl/1.html",
		Timeout:    100,
	}
	r := requesterMock{
		requestCounter: 0,
		returnsError:   true,
	}
	cr := NewCrawler(&r)
	ctx := context.Background()
	// ctx, _ = context.WithTimeout(ctx, time.Second*1)
	go cr.Scan(ctx, cfg.Url, cfg.MaxDepth)
	var err error
	select {
	case <-ctx.Done():
		require.True(t, false)
		return
	case msg := <-cr.ChanResult():
		err = msg.Err
	}
	require.NotNil(t, err)
	assert.Equal(t, err.Error(), "test error")
}

func TestScanCheckChangingDepth(t *testing.T) {
	cfg := Config{
		MaxDepth: 1,
		// crawler works recursive, thats why we get only "http://testUrl/1.html"
		// if we set maxdepth = 1
		// but we add 2 more of depth, so we got 3 results
		// because pageStube.get() returns
		// [firstlink]
		// [firstlink, secondlink]
		// [firstlink, secondlink, thirdlink]
		MaxResults: 3,
		MaxErrors:  1,
		Url:        "http://testUrl/1.html",
		Timeout:    100,
	}
	r := requesterMock{
		requestCounter: 0,
		returnsError:   false,
	}
	cr := NewCrawler(&r)
	ctx := context.Background()
	// ctx, _ = context.WithTimeout(ctx, time.Second*1)
	go cr.Scan(ctx, cfg.Url, cfg.MaxDepth)
	res := []string{}
	_ = res
	cr.ChangeDepth(2)
	func() {
		for cfg.MaxResults > 0{
			select {
			case <-ctx.Done():
				require.True(t, false)
				return
			case msg := <-cr.ChanResult():
				// we have to send signal to increment depth, so
				// we want error first and if we get it we send signal
				if cfg.MaxResults > 0 {
					cfg.MaxResults--
					res = append(res, msg.Title)
				} else {
					return
				}
			}
		}
	}()
	assert.NotEqual(t, res, "")
	assert.Equal(t, 3, len(cr.visited))
}
