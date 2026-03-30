package db

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gstark/agent-manager/internal/config"
)

func packPath(name string) string {
	return filepath.Join(config.PacksDir(), name+".toml")
}

func SavePack(p *Pack) error {
	f, err := os.Create(packPath(p.Name))
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(p)
}

func LoadPack(name string) (*Pack, error) {
	p := &Pack{}
	if _, err := toml.DecodeFile(packPath(name), p); err != nil {
		return nil, err
	}
	return p, nil
}

func ListPacks() ([]*Pack, error) {
	entries, err := os.ReadDir(config.PacksDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var packs []*Pack
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".toml")
		p, err := LoadPack(name)
		if err != nil {
			continue
		}
		packs = append(packs, p)
	}
	return packs, nil
}

func DeletePack(name string) error {
	return os.Remove(packPath(name))
}
