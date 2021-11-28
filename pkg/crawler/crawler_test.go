package crawler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"lesson1/pkg/config"
	"lesson1/pkg/page"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

)

type requesterMock struct {
	requestCounter int // to cunt number of requests
	returnsError   bool
}

func (r *requesterMock) Get(ctx context.Context, url string) (page.Page, error) {
	r.requestCounter += 1
	if r.returnsError {
		r.returnsError = false
		return nil, fmt.Errorf("test error")
	}
	ps := pageMock{
		requestCounter: r.requestCounter,
	}
	return &ps, nil
}

type pageMock struct {
	requestCounter int // sent from requester
}

func (p *pageMock) GetTitle(ctx context.Context) string {
	// as crawler uses interface that's the way to send counter to test function
	return fmt.Sprintf("stubPageTitle%d", p.requestCounter)
}

func (p *pageMock) GetLinks(ctx context.Context) []string {

	return []string{
		"firstlink",
		"secondlink",
		"thirdlink",
		"fourthlink",
		"fifthlink",
		"sixthlink",
	}[:p.requestCounter]
}

func TestScanDepth1(t *testing.T) {
	cfg := config.Config{
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
	cfg := config.Config{
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
	cfg := config.Config{
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
	cfg := config.Config{
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
