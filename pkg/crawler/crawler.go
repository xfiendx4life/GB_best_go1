package crawler

import "context"

type CrawlResult struct {
	Err   error
	Title string
	Url   string
}

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth int)
	ChanResult() <-chan CrawlResult
	ChangeDepth(int)
}
