package sheet

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"tg_seller/internal/model"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetService struct {
	Base64Creds   string
	SpreadsheetID string
	SheetID       string
	SheetName     string
	PauseMs       int // пауза между запросами в миллисекундах
	srv           *sheets.Service
	limiterMu     sync.Mutex
	lastCall      time.Time
	colMap        ColumnMap
}

type ColumnMap map[string]int // например: "N": 0, "Name": 1, ...

// Создает ColumnMap по умолчанию (жестко заданный порядок)
func NewDefaultColumnMap() ColumnMap {
	return ColumnMap{
		"N":              0,
		"Name":           1,
		"Phone":          2,
		"Bar":            3,
		"RegistrationAt": 4,
		"Username":       5,
	}
}

// Создает ColumnMap из строки порядка (например: "N,Name,Phone,Bar,RegistrationAt,Username")
func CreateColumnMapFromOrder(order string) ColumnMap {
	if order == "" {
		return NewDefaultColumnMap()
	}
	fields := strings.Split(order, ",")
	m := make(ColumnMap)
	for idx, field := range fields {
		m[strings.TrimSpace(field)] = idx
	}
	return m
}

// Конструктор SheetService
func NewSheetService(base64Creds, spreadsheetID, sheetID string, pauseMs int, colMap ColumnMap) (*SheetService, error) {
	ctx := context.Background()
	credBytes, err := base64.StdEncoding.DecodeString(base64Creds)
	if err != nil {
		return nil, fmt.Errorf("не удается декодировать credentials из base64: %v", err)
	}
	creds, err := google.CredentialsFromJSON(ctx, credBytes, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("не удается создать credentials из JSON: %v", err)
	}
	srv, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("не удается инициализировать сервис Google Sheets: %v", err)
	}

	s := &SheetService{
		Base64Creds:   base64Creds,
		SpreadsheetID: spreadsheetID,
		SheetID:       sheetID,
		PauseMs:       pauseMs,
		srv:           srv,
		lastCall:      time.Now(),
		colMap:        colMap,
	}

	// Получаем имя листа
	err = s.fetchSheetName()
	if err != nil {
		return nil, fmt.Errorf("не удается получить имя листа: %v", err)
	}

	return s, nil
}

// Добавляем метод для получения имени листа
func (s *SheetService) fetchSheetName() error {
	s.Wait() // лимитер

	resp, err := s.srv.Spreadsheets.Get(s.SpreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("ошибка получения информации о таблице: %v", err)
	}

	for _, sheet := range resp.Sheets {
		if fmt.Sprint(sheet.Properties.SheetId) == s.SheetID {
			s.SheetName = sheet.Properties.Title
			return nil
		}
	}

	return fmt.Errorf("лист с ID %s не найден", s.SheetID)
}

// Лимитер: вызывает паузу между запросами
func (s *SheetService) Wait() {
	s.limiterMu.Lock()
	defer s.limiterMu.Unlock()
	elapsed := time.Since(s.lastCall)
	pause := time.Duration(s.PauseMs) * time.Millisecond
	if elapsed < pause {
		time.Sleep(pause - elapsed)
	}
	s.lastCall = time.Now()
}

// Вставка клиента в таблицу
func (s *SheetService) InsertClient(row int, client model.Client) error {
	s.Wait() // лимитер

	values := make([]interface{}, len(s.colMap))
	for field, idx := range s.colMap {
		switch field {
		case "N":
			values[idx] = client.ID
		case "Name":
			values[idx] = client.Name
		case "Phone":
			values[idx] = client.Phone
		case "Bar":
			values[idx] = client.Bar
		case "RegistrationAt":
			values[idx] = client.RegistrationAt
		case "Username":
			values[idx] = client.Username
		}
	}

	vr := &sheets.ValueRange{
		Values: [][]interface{}{values},
	}

	// Используем имя листа вместо ID
	rangeStr := fmt.Sprintf("%s!A%d", s.SheetName, row)
	_, err := s.srv.Spreadsheets.Values.Update(s.SpreadsheetID, rangeStr, vr).ValueInputOption("RAW").Do()
	if err != nil {
		return fmt.Errorf("ошибка вставки в таблицу: %w", err)
	}
	return nil
}

// FindFirstFreeRow ищет первую свободную строку (из первых 6 строк), где все ячейки пустые (учитывает пробелы)
func (s *SheetService) FindFirstFreeRow() (int, error) {
	s.Wait() // лимитер

	colCount := 6
	// Используем имя листа вместо ID
	rangeStr := fmt.Sprintf("%s!A1:F6", s.SheetName)
	resp, err := s.srv.Spreadsheets.Values.Get(s.SpreadsheetID, rangeStr).Do()
	if err != nil {
		return 0, fmt.Errorf("ошибка чтения строк: %w", err)
	}

	for i := 0; i < 6; i++ {
		if i >= len(resp.Values) {
			// строка полностью пустая
			return i + 1, nil
		}
		row := resp.Values[i]
		empty := true
		for j := 0; j < colCount; j++ {
			val := ""
			if j < len(row) {
				val = fmt.Sprintf("%v", row[j])
			}
			if strings.TrimSpace(val) != "" {
				empty = false
				break
			}
		}
		if empty {
			return i + 1, nil
		}
	}
	return 0, fmt.Errorf("нет свободных строк в первых 6 строках")
}
