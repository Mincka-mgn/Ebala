package main

import (
	"fmt"
	"strings"
)

func tempCraneWheelsCount(s TempSelection) int {
	if n, ok := s.Params["draft_count"].(int); ok && n > 0 {
		return n
	}
	if n := len(s.Wheels); n > 0 {
		return n
	}
	if v, ok := s.Params["wheel_count"].(int); ok {
		return v
	}
	return 0
}
func tempCraneForcesText(s TempSelection, tons bool) string {
	if len(s.Wheels) > 0 {
		return tempForcesUnit(s, tons)
	}
	if raw, ok := s.Params["draft_forces"].(string); ok && raw != "" {
		if !tons {
			return raw
		}
		vals := strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == ' ' || r == '\n' || r == '\t' })
		out := make([]string, 0, len(vals))
		for _, v := range vals {
			n, e := parseNumber(v)
			if e != nil {
				return raw
			}
			out = append(out, tempRU(n/9.80665, 3))
		}
		return strings.Join(out, "; ")
	}
	return ""
}
func tempCraneStepsText(s TempSelection) string {
	if v := tempSteps(s); v != "" {
		return v
	}
	if raw, ok := s.Params["draft_steps"].(string); ok {
		return raw
	}
	return ""
}
func tempCraneSetCell(s *TempSelection, col int, val string, tons bool) error {
	if s.Params == nil {
		s.Params = map[string]interface{}{}
	}
	if col == 1 {
		if val != "" {
			s.Title = val
		}
		return nil
	}
	n := tempCraneWheelsCount(*s)
	forces := tempCraneForcesText(*s, false)
	steps := tempCraneStepsText(*s)
	switch col {
	case 2:
		if val == "" {
			n = 0
		} else {
			v, e := parseNumber(val)
			if e != nil || v != float64(int(v)) || v < 1 || v > 64 {
				return fmt.Errorf("Число колёс должно быть целым от 1 до 64")
			}
			n = int(v)
		}
	case 3:
		if tons && val != "" {
			raw := strings.FieldsFunc(val, func(r rune) bool { return r == ';' || r == ' ' || r == '\n' || r == '\t' })
			vals := make([]string, 0, len(raw))
			for _, item := range raw {
				x, e := parseNumber(item)
				if e != nil {
					return fmt.Errorf("Неверная нагрузка колеса: %s", item)
				}
				vals = append(vals, tempRU(x*9.80665, 5))
			}
			forces = strings.Join(vals, "; ")
		} else {
			forces = val
		}
	case 4:
		steps = val
	default:
		return nil
	}
	s.Params["draft_count"] = n
	s.Params["draft_forces"] = forces
	s.Params["draft_steps"] = steps
	if n == 0 || forces == "" {
		s.Wheels = nil
		return nil
	}
	built, e := tempManualSteps(s.Title, n, forces, steps)
	if e != nil {
		s.Wheels = nil
		return nil
	} // allow partially filled row, not exported until valid
	s.Wheels = built.Wheels
	if s.Source == "" || s.Source == "Данные пользователя" {
		s.Group = "Краны · ручной ввод"
		s.Source = "Данные пользователя"
		s.Status = "user_defined"
	} else {
		s.Status = "modified_by_user"
	}
	return nil
}
