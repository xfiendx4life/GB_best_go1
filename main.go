package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type CrawlResult struct {
	Err   error
	Title string
	Url   string
}

type Page interface {
	GetTitle(context.Context) string
	GetLinks(context.Context) []string
}

type page struct {
	doc *goquery.Document
}

func NewPage(raw io.Reader) (Page, error) {
	doc, err := goquery.NewDocumentFromReader(raw)
	if err != nil {
		return nil, err
	}
	return &page{doc: doc}, nil
}

func (p *page) GetTitle(ctx context.Context) string {
	select {
	case <-ctx.Done():
		return ""
	default:
		return p.doc.Find("title").First().Text()
	}
}

func (p *page) GetLinks(ctx context.Context) []string {
	select {
	case <-ctx.Done():
		return nil
	default:
		var urls []string
		p.doc.Find("a").Each(func(_ int, s *goquery.Selection) {
			url, ok := s.Attr("href")
			if ok {
				urls = append(urls, url)
			}
		})
		return urls
	}
}

type Requester interface {
	Get(ctx context.Context, url string) (Page, error)
}

type requester struct {
	timeout time.Duration
}

func NewRequester(timeout time.Duration) requester {
	return requester{timeout: timeout}
}

func (r requester) Get(ctx context.Context, url string) (Page, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		cl := &http.Client{
			Timeout: r.timeout,
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		body, err := cl.Do(req)
		if err != nil {
			return nil, err
		}
		defer body.Body.Close()
		page, err := NewPage(body.Body)
		if err != nil {
			return nil, err
		}
		return page, nil
	}
	return nil, nil
}

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth int)
	ChanResult() <-chan CrawlResult
	ChangeDepth(int)
}

type crawler struct {
	r         Requester
	res       chan CrawlResult
	chngDepth chan int // для изменения глубины поиска
	visited   map[string]struct{}
	mu        sync.RWMutex
}

func NewCrawler(r Requester) *crawler {
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

//Config - структура для конфигурации
type Config struct {
	MaxDepth   int
	MaxResults int
	MaxErrors  int
	Url        string
	Timeout    int //in seconds
}

func mainStarter() {
	cfg := Config{
		MaxDepth:   3,
		MaxResults: 10,
		MaxErrors:  10,
		Url:        "http://telegram.org",
		Timeout:    3,
	}
	var cr Crawler

	r := NewRequester(time.Duration(cfg.Timeout) * time.Second)
	cr = NewCrawler(r)

	ctx, _ := context.WithCancel(context.Background())
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cfg.Timeout)) // добавим таймаут в контекст
	go cr.Scan(ctx, cfg.Url, cfg.MaxDepth)                                          //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg)                                          //Обрабатываем результаты в отдельной рутине

	sigCh := make(chan os.Signal, 1)                      //Создаем канал для приема сигналов
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGUSR1) //Подписываемся на сигнал SIGINT
	for {
		select {
		case <-ctx.Done(): //Если всё завершили - выходим
			log.Print("ended with context")
			return
		case sign := <-sigCh:
			switch sign {
			case syscall.SIGINT:
				cancel() //Если пришёл сигнал SigInt - завершаем контекст
			case syscall.SIGUSR1:
				cr.ChangeDepth(2)
			}
		}
	}
}

func main() {
	mainStarter()
}

func processResult(ctx context.Context, cancel func(), cr Crawler, cfg Config) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cr.ChanResult():
			if msg.Err != nil {
				maxErrors--
				log.Printf("crawler result return err: %s\n", msg.Err.Error())
				syscall.Kill(os.Getpid(), syscall.SIGUSR1)
				if maxErrors <= 0 {
					cancel()
					return
				}
			} else {
				maxResult--
				log.Printf("crawler result: [url: %s] Title: %s\n", msg.Url, msg.Title)
				if maxResult <= 0 {
					cancel()
					return
				}
			}
		}
	}
}
