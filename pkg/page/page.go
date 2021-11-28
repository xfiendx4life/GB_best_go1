package page

import "context"

type Page interface {
	GetTitle(context.Context) string
	GetLinks(context.Context) []string
}
