package crawler

import (
	// File is not `gofmt`-ed with `-s` (gofmt) "sync"- changed imports order, put sync after lesson1/pkg/requester
	"context"
	"lesson1/pkg/requester"
	"sync"
)

type crawler struct {
	r         requester.Requester
	res       chan CrawlResult
	chngDepth chan int // для изменения глубины поиска
	visited   map[string]struct{}
	mu        sync.RWMutex
}

func NewCrawler(r requester.Requester) *crawler {
	return &crawler{
		r:         r,
		res:       make(chan CrawlResult),
		chngDepth: make(chan int), // для изменения глубины поиска
		visited:   make(map[string]struct{}),
		mu:        sync.RWMutex{},
	}
}

func (c *crawler) Scan(ctx context.Context, url string, depth int) {
	if depth <= 0 { //Проверяем то, что есть запас по глубине
		return
	}
	c.mu.RLock()
	_, ok := c.visited[url] //Проверяем, что мы ещё не смотрели эту страницу
	c.mu.RUnlock()
	if ok {
		return
	}
	select {
	case <-ctx.Done(): //Если контекст завершен - прекращаем выполнение
		return
	case d := <-c.chngDepth:
		go c.Scan(ctx, url, depth+d)
		return
	default:
		page, err := c.r.Get(ctx, url) //Запрашиваем страницу через Requester
		if err != nil {
			c.res <- CrawlResult{Err: err} //Записываем ошибку в канал
			return
		}
		c.mu.Lock()
		c.visited[url] = struct{}{} //Помечаем страницу просмотренной
		c.mu.Unlock()
		c.res <- CrawlResult{ //Отправляем результаты в канал
			Title: page.GetTitle(ctx),
			Url:   url,
		}
		for _, link := range page.GetLinks(ctx) {
			go c.Scan(ctx, link, depth-1) //На все полученные ссылки запускаем новую рутину сборки
		}
	}
}

func (c *crawler) ChanResult() <-chan CrawlResult {
	return c.res
}

func (c *crawler) ChangeDepth(val int) {
	c.chngDepth <- val
}
