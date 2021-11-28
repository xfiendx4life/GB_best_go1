package page

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
