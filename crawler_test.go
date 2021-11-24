package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	// "fmt"
	"io/ioutil"
	"log"
	"os"

	// "sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type crawlerStub struct {
	// r         requesterMock
	res       chan CrawlResult
	chngDepth chan int // для изменения глубины поиска
	// visited   map[string]struct{}
	// mu        sync.RWMutex
	testFunc func() []CrawlResult
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
	var err error
	func() {
		for cfg.MaxResults > 0 {
			select {
			case <-ctx.Done():
				require.True(t, false)
				return
			case <-cr.ChanResult():
				if cfg.MaxResults > 0 {
					cfg.MaxResults--
				} else {
					return
				}
			}
		}
	}()
	assert.Nil(t, err)
	assert.Equal(t, 3, len(cr.visited))
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
		for cfg.MaxResults > 0 {
			select {
			case <-ctx.Done():
				require.True(t, false)
				return
			case msg := <-cr.ChanResult():
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

func createReader() io.Reader {
	r, err := os.ReadFile("1.html")
	if err != nil {
		log.Fatal("can't read test html file")
	}
	return strings.NewReader(string(r))
}

func TestPageGetTitle(t *testing.T) {
	p, err := NewPage(createReader())
	assert.Nil(t, err)
	ctx := context.Background()
	assert.Equal(t, "Document", p.GetTitle(ctx))
}

func TestGeLinks(t *testing.T) {
	p, err := NewPage(createReader())
	assert.Nil(t, err)
	links := p.GetLinks(context.Background())
	assert.Equal(t, 3, len(links))
}

func startLocalServer(ctx context.Context) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.ParseFiles("1.html")
		t.Execute(w, struct{}{})
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}


func TestRequesterGet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go startLocalServer(ctx)
	r := NewRequester(time.Duration(3) * time.Second)
	p, err := r.Get(ctx, "http://localhost:8080")
	assert.Nil(t, err)
	assert.NotNil(t, p)
	cancel()
}

func TestRequesterGetNoServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// go startLocalServer(ctx)
	r := NewRequester(time.Duration(3) * time.Second)
	_, err := r.Get(ctx, "http://localhost:8000")
	assert.NotNil(t, err)
	cancel()
}
