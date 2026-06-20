package config

import (
	"encoding/xml"
	"os"
	"strings"
)

type EReadersConfig struct {
	XMLName  xml.Name       `xml:"EasyPubConfig"`
	EReaders []EReaderModel `xml:"eReaders>model"`
}

type EReaderModel struct {
	Name  string   `xml:"name"`
	Fonts []string `xml:"font"`
}

func LoadEReaders(path string) (*EReadersConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg EReadersConfig
	if err := xml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *EReadersConfig) Model(index int) (EReaderModel, bool) {
	if c == nil || index < 0 || index >= len(c.EReaders) {
		return EReaderModel{}, false
	}
	return c.EReaders[index], true
}

func (r RecentOptions) CustomizedFonts() []string {
	return splitFontList(r.FontCustomized)
}

func (r RecentOptions) MachineIDInt() int {
	return intDefault(r.MachineID, -1)
}

func splitFontList(value string) []string {
	fields := strings.FieldsFunc(value, func(ch rune) bool {
		return ch == '\n' || ch == '\r' || ch == ';' || ch == '|'
	})
	var out []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}
