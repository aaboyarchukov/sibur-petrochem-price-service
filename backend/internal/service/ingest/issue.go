// Package ingest — парсинг пользовательских .xlsx (ssp, formulas) в доменные строки.
// Валидация строгая: любая проблема отклоняет файл целиком, БД не трогается.
package ingest

import "strconv"

// Issue — одна проблема валидации файла.
type Issue struct {
	Row    int    // номер строки листа (1 — заголовки); 0 — проблема уровня файла
	Column string // имя колонки; пусто — проблема всей строки или файла
	Reason string
}

func (i Issue) String() string {
	out := i.Reason
	if i.Column != "" {
		out = "колонка «" + i.Column + "»: " + out
	}
	if i.Row > 0 {
		out = "строка " + strconv.Itoa(i.Row) + ", " + out
	}

	return out
}
