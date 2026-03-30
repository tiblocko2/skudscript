// Приложение для отметки сотрудников СКУД
// Считывает карту через считыватель и создает Excel отчеты
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"skudscript/db"

	"github.com/xuri/excelize/v2"
)

// readCardID считывает ID карты с клавиатуры (считывателя)
// Ожидает ввод и нажатие Enter
func readCardID(reader *bufio.Reader) (string, error) {
	fmt.Println("\n📌 Приложите карту к считывателю...")
	fmt.Print("> ")

	cardID, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("ошибка чтения карты: %w", err)
	}

	// Удаляем символы новой строки и пробелы
	cardID = strings.TrimSpace(cardID)

	if cardID == "" {
		return "", fmt.Errorf("карта не считана")
	}

	return cardID, nil
}

// getReportDir возвращает путь к папке для хранения отчетов
// Папка Otchet создается в той же директории, что и exe файл
func getReportDir(exePath string) string {
	return filepath.Join(filepath.Dir(exePath), "Otchet")
}

// getReportFilename возвращает имя файла отчета для указанной даты
// Формат: YYYY-MM-DD.xlsx
func getReportFilename(date time.Time) string {
	return date.Format("2006-01-02") + ".xlsx"
}

// getReportPath возвращает полный путь к файлу отчета
func getReportPath(exePath string, date time.Time) string {
	reportDir := getReportDir(exePath)
	return filepath.Join(reportDir, getReportFilename(date))
}

// ensureReportDir создает папку для отчетов, если она не существует
func ensureReportDir(exePath string) error {
	reportDir := getReportDir(exePath)
	return os.MkdirAll(reportDir, 0755)
}

// createNewReport создает новый Excel файл отчета для указанной даты
// Добавляет заголовки столбцов: Время, ФИО, Карточка
func createNewReport(exePath string, date time.Time) error {
	// Создаем папку для отчетов, если она не существует
	err := ensureReportDir(exePath)
	if err != nil {
		return fmt.Errorf("ошибка создания папки отчетов: %w", err)
	}

	reportPath := getReportPath(exePath, date)

	// Создаем новый Excel файл
	f := excelize.NewFile()
	defer f.Close()

	// Получаем имя листа по умолчанию
	sheetName := f.GetSheetName(0)

	// Устанавливаем заголовки столбцов
	headers := []string{"Время", "ФИО", "Карточка"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Настраиваем ширину столбцов
	f.SetColWidth(sheetName, "A", "A", 15) // Время
	f.SetColWidth(sheetName, "B", "B", 30) // ФИО
	f.SetColWidth(sheetName, "C", "C", 20) // Карточка

	// Делаем заголовки жирными
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	for i := 1; i <= len(headers); i++ {
		cell, _ := excelize.CoordinatesToCellName(i, 1)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	// Сохраняем файл
	if err := f.SaveAs(reportPath); err != nil {
		return fmt.Errorf("ошибка сохранения отчета: %w", err)
	}

	return nil
}

// checkReportExists проверяет, существует ли файл отчета для указанной даты
func checkReportExists(exePath string, date time.Time) bool {
	reportPath := getReportPath(exePath, date)
	_, err := os.Stat(reportPath)
	return err == nil
}

// addRecordToReport добавляет запись в Excel отчет
// Создает новый отчет, если он еще не существует
func addRecordToReport(exePath string, date time.Time, cardID, fullName string) error {
	reportPath := getReportPath(exePath, date)

	// Если отчет не существует, создаем новый
	if !checkReportExists(exePath, date) {
		err := createNewReport(exePath, date)
		if err != nil {
			return err
		}
	}

	// Открываем существующий файл
	f, err := excelize.OpenFile(reportPath)
	if err != nil {
		return fmt.Errorf("ошибка открытия отчета: %w", err)
	}
	defer f.Close()

	// Получаем имя листа
	sheetName := f.GetSheetName(0)

	// Находим последнюю заполненную строку
	// Получаем все строки листа для определения последней заполненной
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("ошибка чтения строк: %w", err)
	}
	lastRow := len(rows)
	nextRow := lastRow + 1

	// Форматируем текущее время
	currentTime := time.Now().Format("15:04:05")

	// Добавляем новую строку с данными
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", nextRow), currentTime)
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", nextRow), fullName)
	f.SetCellValue(sheetName, fmt.Sprintf("C%d", nextRow), cardID)

	// Сохраняем файл
	if err := f.SaveAs(reportPath); err != nil {
		return fmt.Errorf("ошибка сохранения отчета: %w", err)
	}

	return nil
}

// getExePath возвращает путь к текущему исполняемому файлу
func getExePath() string {
	exePath, err := os.Executable()
	if err != nil {
		// Если не удалось получить путь к exe, используем текущую директорию
		exePath, _ = os.Getwd()
	}
	return exePath
}

func main() {
	// Настройка консоли для корректного отображения UTF-8 на Windows
	cmd := exec.Command("chcp", "65001")
	cmd.Run()

	fmt.Println("\n🚀 Система отметки сотрудников СКУД")
	fmt.Println(strings.Repeat("=", 50))

	// Инициализация базы данных
	database, err := db.InitDB(getExePath())
	if err != nil {
		fmt.Printf("❌ Ошибка инициализации базы данных: %v\n", err)
		fmt.Println("\nНажмите Enter для выхода...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}
	defer database.Close()

	reader := bufio.NewReader(os.Stdin)

	// Считывание карты
	cardID, err := readCardID(reader)
	if err != nil {
		fmt.Printf("❌ Ошибка считывания карты: %v\n", err)
		fmt.Println("\nНажмите Enter для выхода...")
		reader.ReadString('\n')
		return
	}

	// Поиск сотрудника по карте
	user, err := database.GetUserByCardID(cardID)
	if err != nil {
		fmt.Printf("❌ Ошибка поиска сотрудника: %v\n", err)
		fmt.Println("\nНажмите Enter для выхода...")
		reader.ReadString('\n')
		return
	}

	if user == nil {
		fmt.Println("\n❌ Сотрудник с такой картой не найден!")
		fmt.Println("   Обратитесь к администратору для регистрации карты.")
		fmt.Println("\nНажмите Enter для выхода...")
		reader.ReadString('\n')
		return
	}

	// Получаем текущую дату
	currentDate := time.Now()

	// Добавляем запись в отчет
	err = addRecordToReport(getExePath(), currentDate, user.CardID, user.FullName)
	if err != nil {
		fmt.Printf("❌ Ошибка записи в отчет: %v\n", err)
		fmt.Println("\nНажмите Enter для выхода...")
		reader.ReadString('\n')
		return
	}

	// Успешная отметка
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ Отметка успешна!")
	fmt.Printf("   Сотрудник: %s\n", user.FullName)
	fmt.Printf("   Время: %s\n", time.Now().Format("15:04:05"))
	fmt.Printf("   Дата: %s\n", currentDate.Format("02.01.2006"))
	fmt.Println(strings.Repeat("=", 50))

	fmt.Println("\nНажмите Enter для выхода...")
	reader.ReadString('\n')
}
