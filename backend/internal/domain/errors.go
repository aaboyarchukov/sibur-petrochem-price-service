// Package domain — доменные модели и sentinel-ошибки сервиса расчёта цен.
package domain

import "errors"

var (
	// ErrNotImplemented — операция ещё не реализована (скелет сервиса).
	ErrNotImplemented = errors.New("not implemented")
	// ErrCalculationNotFound — расчёт с таким id не существует.
	ErrCalculationNotFound = errors.New("calculation not found")
	// ErrRowNotFound — строка расчёта не найдена.
	ErrRowNotFound = errors.New("row not found")
	// ErrSourceNotFound — источник с таким ключом не найден.
	ErrSourceNotFound = errors.New("source not found")
	// ErrFormulaNotAllowed — формула не входит в список подходящих для строки.
	ErrFormulaNotAllowed = errors.New("formula not in row candidates")
	// ErrInvalidPrice — ручная цена не проходит валидацию (не число / <= 0).
	ErrInvalidPrice = errors.New("invalid manual price")
	// ErrAlreadySubmitted — участок уже присоединён к сводному документу.
	ErrAlreadySubmitted = errors.New("part already submitted")
	// ErrSourcesNotLoaded — источники не загружены, расчёт невозможен.
	ErrSourcesNotLoaded = errors.New("sources not loaded")
	// ErrComponent — не удалось разрешить значение компонента формулы.
	ErrComponent = errors.New("component resolution error")
)
