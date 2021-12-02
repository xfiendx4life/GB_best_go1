package main

import (
	"context"
	"lesson1/pkg/config"
	"lesson1/pkg/crawler"
	"lesson1/pkg/process"
	"lesson1/pkg/requester"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func mainStarter() {
	cfg := config.Config{}
	err := cfg.ReadConfigFromFile(os.Getenv("CONFIGPATH"))
	if err != nil {
		log.Fatalf("can't create config %s", err)
	}
	var cr crawler.Crawler

	r := requester.NewRequester(time.Duration(cfg.Timeout) * time.Second)
	cr = crawler.NewCrawler(r)

	// lostcancel: the cancel function returned by context.WithCancel should be called, not discarded, to avoid a context leak (govet)
	// defer cancel func to close context in any case
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(cfg.Timeout)) // добавим таймаут в контекст
	go cr.Scan(ctx, cfg.Url, cfg.MaxDepth)                                         //Запускаем краулер в отдельной рутине
	go process.ProcessResult(ctx, cancel, cr, cfg)                                 //Обрабатываем результаты в отдельной рутине

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
