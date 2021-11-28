package requester

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func startLocalServer(ctx context.Context) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.ParseFiles("1.html")
		t.Execute(w, struct{}{})
	})
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func TestRequesterGet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go startLocalServer(ctx)
	r := NewRequester(time.Duration(3) * time.Second)
	p, err := r.Get(ctx, "http://localhost:8000")
	assert.Nil(t, err)
	assert.NotNil(t, p)
	cancel()
}

func TestRequesterGetNoServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// go startLocalServer(ctx)
	r := NewRequester(time.Duration(3) * time.Second)
	_, err := r.Get(ctx, "http://localhost:2021")
	assert.NotNil(t, err)
	cancel()
}
