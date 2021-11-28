package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func (c *Config) ReadConfigFromFile(path string) (err error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("can't open file %s", err)
	}
	err = json.Unmarshal(f, c)
	if err != nil {
		return fmt.Errorf("can't unmarshal json %s", err)
	}
	return nil
}
