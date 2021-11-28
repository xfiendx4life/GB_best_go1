package config

//Config - структура для конфигурации
type Config struct {
	MaxDepth   int    `json:"maxdepth"`
	MaxResults int    `json:"maxresults"`
	MaxErrors  int    `json:"maxerrors"`
	Url        string `json:"url"`
	Timeout    int    `json:"timeout"` //in seconds
}
