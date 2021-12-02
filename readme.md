# Юнит тест краулера

Для начала попробовал написать стабы для всех интерфейсов без дополнительного функционала. 

+ Начал тетирования от общего - `TestProcessResult`.
Тут я использовал `crawlerStub` просто выдавая хардкод результаты ссылок. Проблемой оказалось тестировать вывод, но, выход нашелся в перенаправлении вывода в файл.
```go
r, w, _ := os.Pipe() 
log.SetOutput(w)
data, _ := ioutil.ReadAll(r)
...
```
Также использовал мок метода `Scan`, просто отсылая в канал заготовленные "ссылки":
```go
func (c *crawlerStub) Scan(ctx context.Context, url string, depth int) {
	for _, result := range c.testFunc() {
		c.res <- result
	}
}
```
Проверил на содержание ссылок в файле с выводом.

+ Проверил коррекнтный возврат ошибок в `TestProcessResultError`. Ошибки получал также из файла.

+ Проверил выполнение отмены через контекст, для достижения максимального количества ошибок и сообщений. Для этого обернул `cancelFunc` в функциях `TestProcessResultMaxErrorsCancelFunc` и `TestProcessResultMaxResultsCancelFunc` проверил вызов функции:
```go
processResult(ctx, func() {
    cancel()
    *cancePointer = true
}, &c, cfg)
require.True(t, cancelChecker)
```
+ В провеках глубины за счет особенностей хардкода нужно выставлять `MaxResults` равную глубине. Это сделано в `TestScanDepth1`, `TestScanCheckDepth`. В последней проверяю количество обработанных ссылок

+ В `TestScanCheckChangingDepth` проверяю изменение глубины поиска, с 1 до 3, получаю три ссылки. За счет реализации `pageStub.GetLinks` 
```go
func (p *pageStub) GetLinks(ctx context.Context) []string {

	return []string{
		"firstlink",
		"secondlink",
		"thirdlink",
		"fourthlink",
		"fifthlink",
		"sixthlink",
	}[:p.requestCounter]
}
```
Количество полученных ссылок будет равно глубине.

+ Двигаюсь к простейшим реализациям. Для теста `TestPageGetTitle` и `TestGeLinks` вместо станицы полученной через реквест подсовываю объекту `page` `strings.Reader`, который реадизует интерфейс `io.Reader` с текстом страницы из файла `1.txt`. 

+ Для проверки объекта `requester` сделал простейший сервер и рендерю в нем  по запросу шаблон страницы `1.txt`
```go
func startLocalServer(ctx context.Context) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.ParseFiles("1.html")
		t.Execute(w, struct{}{})
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
Делаю к нему запрос и проверяю результаты:
```go
r := NewRequester(time.Duration(3) * time.Second)
p, err := r.Get(ctx, "http://localhost:8080")
assert.Nil(t, err)
assert.NotNil(t, p)
```

## UPD. Изменена структура файлов. 
+ Изменена структура файлов. Попробовал сделать гексагональную структуру, насколько понял.
+ Тесты разобрал по пакетам согласно их принадлежности 
+ Добавил чтение конфига из JSON-файла. 
Для чтения из правильного файла путь нужно указать используя переменную окружения:
```bash
CONFIGPATH=../../config.json go run .
```
Выше пример файла с моим конфигом.


## Задание 6

 + Добавлена конфигурация для линтеров (на основе методички и конфигурации с сайта golangci-lint)
 + Выбраны линтеры, кроме обязательных:
  	- goconst # нахождение строк, которые следует вынести в константы
  	- funlen # детектирование слишком крупных функций
	- errcheck # проверка на обработку всех ошибок
	- deadcode # детектирование не использованного кода
	- gochecknoglobals # поиск глобальных переменных
 + Проверка в тестах отменена
 + найдены ошибки:
 ```go
 pkg/requester/service.go:42: File is not `gofmt`-ed with `-s` (gofmt)
}
pkg/crawler/service.go:5: File is not `gofmt`-ed with `-s` (gofmt)
        "sync"
cmd/crawler/main.go:27:7: lostcancel: the cancel function returned by context.WithCancel should be called, not discarded, to avoid a context leak (govet)
        ctx, _ := context.WithCancel(context.Background())
             ^
pkg/requester/service.go:41:2: unreachable: unreachable code (govet)
        return nil, nil
```
 + проблемы решены:
	- В проблемах с форматированием, добавлена пустая строка и изменен порядок импортов на алфавитный, соотевтственно. 
	- С cancel() добавлен `defer cancel()` для первой функции, чтобы отменять контекст в любом случае
	- Удален ненужный `return nil, nil`
 + Места проблем, решенных линтерами помечены комментарием в коде