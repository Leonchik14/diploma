package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type hhAreaNode struct {
	ID       string       `json:"id"`
	ParentID *string      `json:"parent_id"`
	Name     string       `json:"name"`
	Areas    []hhAreaNode `json:"areas"`
}

func loadAreaNamesCatalog() []string {
	candidates := []string{
		filepath.Clean("hh-data/areas.json"),
		filepath.Clean("../hh-data/areas.json"),
		filepath.Clean("../../hh-data/areas.json"),
		filepath.Clean("../../../hh-data/areas.json"),
	}

	var raw []byte
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			raw = b
			break
		}
	}
	if len(raw) == 0 {
		return nil
	}

	var roots []hhAreaNode
	if err := json.Unmarshal(raw, &roots); err != nil {
		return nil
	}

	items := make([]string, 0, 4096)
	seen := make(map[string]struct{}, 4096)
	var walk func([]hhAreaNode)
	walk = func(nodes []hhAreaNode) {
		for _, n := range nodes {
			name := strings.TrimSpace(n.Name)
			if name != "" {
				if _, ok := seen[name]; !ok {
					seen[name] = struct{}{}
					items = append(items, name)
				}
			}
			if len(n.Areas) > 0 {
				walk(n.Areas)
			}
		}
	}
	walk(roots)

	sort.Strings(items)
	return items
}

