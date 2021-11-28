package page

import (
	"context"
	"io"

	"github.com/PuerkitoBio/goquery"
)

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
