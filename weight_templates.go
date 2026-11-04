package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// A ready-made construction is a named copy of an ordinary weight calculation.
// User entries live in a separate file, independent of built-in suggestions.
type WeightTemplate struct {
	ID   string      `json:"id"`
	Name string      `json:"name"`
	Rows []WeightRow `json:"rows"`
}
type weightTemplateFile struct {
	SchemaVersion int              `json:"schema_version"`
	Items         []WeightTemplate `json:"items"`
}

func templateRoot() string        { return filepath.Join(storageRoot(), "Catalog") }
func templateBuiltInPath() string { return filepath.Join(templateRoot(), "constructions_builtin.json") }
func templateUserPath() string    { return filepath.Join(templateRoot(), "constructions_user.json") }

func readTemplates(path string, optional bool) ([]WeightTemplate, error) {
	b, err := os.ReadFile(path)
	if optional && errors.Is(err, os.ErrNotExist) {
		return []WeightTemplate{}, nil
	}
	if err != nil {
		return nil, err
	}
	var data weightTemplateFile
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("Не удалось прочитать библиотеку конструкций: %w", err)
	}
	if data.SchemaVersion != 1 || len(data.Items) > 500 {
		return nil, fmt.Errorf("Неверный формат библиотеки конструкций")
	}
	for _, item := range data.Items {
		if strings.TrimSpace(item.Name) == "" {
			return nil, fmt.Errorf("В библиотеке есть запись без названия")
		}
		if err := (WeightCalc{Rows: item.Rows}).Validate(); err != nil {
			return nil, fmt.Errorf("Конструкция %q: %w", item.Name, err)
		}
	}
	return data.Items, nil
}
func saveUserTemplates(items []WeightTemplate) error {
	b, err := json.MarshalIndent(weightTemplateFile{SchemaVersion: 1, Items: items}, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(templateUserPath(), append(b, '\n'))
}
func cloneWeightRows(rows []WeightRow) []WeightRow { return append([]WeightRow{}, rows...) }
func saveNamedConstruction(existing []WeightTemplate, name string, rows []WeightRow) ([]WeightTemplate, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return existing, false, fmt.Errorf("Введите название конструкции")
	}
	if len(rows) == 0 {
		return existing, false, fmt.Errorf("Состав конструкции пуст")
	}
	if e := (WeightCalc{Rows: rows}).Validate(); e != nil {
		return existing, false, e
	}
	for i := range existing {
		if strings.EqualFold(existing[i].Name, name) {
			existing[i].Rows = cloneWeightRows(rows)
			return existing, true, nil
		}
	}
	id := "user-1"
	for n := 1; ; n++ {
		id = fmt.Sprintf("user-%d", n)
		ok := true
		for _, e := range existing {
			if e.ID == id {
				ok = false
				break
			}
		}
		if ok {
			break
		}
	}
	return append(existing, WeightTemplate{ID: id, Name: name, Rows: cloneWeightRows(rows)}), false, nil
}
func findTemplateByName(items []WeightTemplate, name string) (WeightTemplate, bool) {
	for _, x := range items {
		if strings.EqualFold(x.Name, name) {
			return x, true
		}
	}
	return WeightTemplate{}, false
}
