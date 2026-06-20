package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEReaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ereaders.xml")
	xml := `<?xml version="1.0" encoding="utf-8"?>
<EasyPubConfig>
  <eReaders>
    <model>
      <name>Reader</name>
      <font>res:///fonts/a.ttf</font>
      <font>res:///fonts/b.ttf</font>
    </model>
  </eReaders>
</EasyPubConfig>`
	if err := os.WriteFile(path, []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadEReaders(path)
	if err != nil {
		t.Fatal(err)
	}
	model, ok := cfg.Model(0)
	if !ok {
		t.Fatal("model 0 not found")
	}
	if model.Name != "Reader" || len(model.Fonts) != 2 {
		t.Fatalf("model = %#v", model)
	}
}
