package requester

import (
	"context"
	"lesson1/pkg/page"
)

type Requester interface {
	Get(ctx context.Context, url string) (page.Page, error)
}
