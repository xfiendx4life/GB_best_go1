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