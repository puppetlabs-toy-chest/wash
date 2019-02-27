package data

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/puppetlabs/wash/plugin"
)

type listableEntry struct {
	entry `yaml:",inline"`
}

func (l *listableEntry) LS(ctx context.Context) ([]plugin.Entry, error) {
	if l.Enum != nil {
		entries := make([]plugin.Entry, len(l.Enum))
		for i, name := range l.Enum {
			entries[i] = newCopy(name, l, l.Proto)
		}
		return entries, nil
	}

	if l.List == "" {
		return nil, fmt.Errorf("enum or list are required for %v", l.Name())
	}

	output, err := invoke(l.List, l)
	if err != nil {
		return nil, err
	}

	entries := make([]plugin.Entry, 0)
	if l.Post != "" {
		// Require structured JSON input
		var data map[string]interface{}
		dec := json.NewDecoder(output)
		if err := dec.Decode(&data); err != nil {
			return nil, err
		}

		t, err := template.New("process list").Parse(l.Post)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err = t.Execute(&buf, data); err != nil {
			return nil, err
		}
		output = &buf
	}

	// Require a list of names
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		entries = append(entries, newCopy(strings.TrimSpace(scanner.Text()), l, l.Proto))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
