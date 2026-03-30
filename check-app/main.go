// Приложение для отметки сотрудников СКУД с графическим интерфейсом
// Считывает карту через считыватель и создает Excel отчеты
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"skudscript/db"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/xuri/excelize/v2"
)

// CheckApp представляет приложение для отметки сотрудников
type CheckApp struct {
	app         fyne.App
	window      fyne.Window
	database    *db.DB
	cardEntry   *widget.Entry
	statusLabel *widget.Label
	infoLabel   *widget.Label
}

// getExePath возвращает путь к текущему исполняемому файлу
func getExePath() string {
	exePath, err := os.Executable()
	if err != nil {
		exePath, _ = os.Getwd()
	}
	return exePath
}

// getReportDir возвращает путь к папке для хранения отчетов
func getReportDir(exePath string) string {
	return filepath.Join(filepath.Dir(exePath), "Otchet")
}

// getReportPath возвращает полный путь к файлу отчета
func getReportPath(exePath string, date time.Time) string {
	reportDir := getReportDir(exePath)
	return filepath.Join(reportDir, date.Format("2006-01-02")+".xlsx")
}

// ensureReportDir создает папку для отчетов, если она не существует
func ensureReportDir(exePath string) error {
	reportDir := getReportDir(exePath)
	return os.MkdirAll(reportDir, 0755)
}

// createNewReport создает новый Excel файл отчета для указанной даты
func createNewReport(exePath string, date time.Time) error {
	err := ensureReportDir(exePath)
	if err != nil {
		return fmt.Errorf("ошибка создания папки отчетов: %w", err)
	}

	reportPath := getReportPath(exePath, date)

	f := excelize.NewFile()
	defer f.Close()

	sheetName := f.GetSheetName(0)

	headers := []string{"Время", "ФИО", "Карточка"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	f.SetColWidth(sheetName, "A", "A", 15)
	f.SetColWidth(sheetName, "B", "B", 30)
	f.SetColWidth(sheetName, "C", "C", 20)

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
func addRecordToReport(exePath string, date time.Time, cardID, fullName string) error {
	reportPath := getReportPath(exePath, date)

	if !checkReportExists(exePath, date) {
		err := createNewReport(exePath, date)
		if err != nil {
			return err
		}
	}

	f, err := excelize.OpenFile(reportPath)
	if err != nil {
		return fmt.Errorf("ошибка открытия отчета: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("ошибка чтения строк: %w", err)
	}
	lastRow := len(rows)
	nextRow := lastRow + 1

	currentTime := time.Now().Format("15:04:05")

	f.SetCellValue(sheetName, fmt.Sprintf("A%d", nextRow), currentTime)
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", nextRow), fullName)
	f.SetCellValue(sheetName, fmt.Sprintf("C%d", nextRow), cardID)

	if err := f.SaveAs(reportPath); err != nil {
		return fmt.Errorf("ошибка сохранения отчета: %w", err)
	}

	return nil
}

// showStatus отображает сообщение о статусе операции
func (a *CheckApp) showStatus(message string, isError bool) {
	a.statusLabel.SetText(message)
	if isError {
		a.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	}
	a.statusLabel.Refresh()
}

// clearStatus очищает сообщение о статусе
func (a *CheckApp) clearStatus() {
	a.statusLabel.SetText("")
}

// resetUI сбрасывает интерфейс в исходное состояние
func (a *CheckApp) resetUI() {
	a.cardEntry.SetText("")
	a.clearStatus()
	a.infoLabel.SetText("")
	a.window.Canvas().Focus(a.cardEntry)
}

// processCard обрабатывает карту сотрудника
func (a *CheckApp) processCard() {
	cardID := strings.TrimSpace(a.cardEntry.Text)

	if cardID == "" {
		a.showStatus("❌ Карта не считана", true)
		return
	}

	// Поиск сотрудника по карте
	user, err := a.database.GetUserByCardID(cardID)
	if err != nil {
		a.showStatus(fmt.Sprintf("❌ Ошибка поиска: %v", err), true)
		return
	}

	if user == nil {
		a.showStatus("❌ Сотрудник с такой картой не найден!\nОбратитесь к администратору", true)
		return
	}

	// Получаем текущую дату
	currentDate := time.Now()

	// Добавляем запись в отчет
	err = addRecordToReport(getExePath(), currentDate, user.CardID, user.FullName)
	if err != nil {
		a.showStatus(fmt.Sprintf("❌ Ошибка записи в отчет: %v", err), true)
		return
	}

	// Успешная отметка
	currentTime := time.Now().Format("15:04:05")
	currentDateStr := currentDate.Format("02.01.2006")

	a.showStatus("✅ Отметка успешна!", false)
	a.infoLabel.SetText(fmt.Sprintf("Сотрудник: %s\nВремя: %s\nДата: %s",
		user.FullName, currentTime, currentDateStr))

	// Автоматический сброс через 3 секунды
	go func() {
		time.Sleep(3 * time.Second)
		a.window.Canvas().Focus(nil)
		a.window.Canvas().Focus(a.cardEntry)
		a.resetUI()
	}()
}

// createUI создает пользовательский интерфейс
func (a *CheckApp) createUI() {
	a.cardEntry = widget.NewEntry()
	a.cardEntry.SetPlaceHolder("Приложите карту к считывателю...")
	a.cardEntry.OnSubmitted = func(s string) {
		a.processCard()
	}

	a.statusLabel = widget.NewLabel("")
	a.statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	a.infoLabel = widget.NewLabel("")
	a.infoLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Иконка карты
	cardIcon := widget.NewIcon(theme.AccountIcon())

	// Инструкция
	instruction := widget.NewLabel("Приложите карту к считывателю")
	instruction.TextStyle = fyne.TextStyle{Bold: true}
	instruction.Alignment = fyne.TextAlignCenter

	// Поле ввода карты (скрытое, для считывателя)
	cardForm := widget.NewForm(
		widget.NewFormItem("ID карты", a.cardEntry),
	)

	// Кнопка ручной обработки
	manualBtn := widget.NewButtonWithIcon("Обработать", theme.ConfirmIcon(), func() {
		a.processCard()
	})
	manualBtn.Importance = widget.HighImportance

	// Кнопка выхода
	exitBtn := widget.NewButtonWithIcon("Выход", theme.CancelIcon(), func() {
		a.app.Quit()
	})

	content := container.NewVBox(
		layout.NewSpacer(),
		cardIcon,
		instruction,
		widget.NewSeparator(),
		container.NewPadded(cardForm),
		container.NewHBox(layout.NewSpacer(), manualBtn, layout.NewSpacer()),
		a.statusLabel,
		a.infoLabel,
		layout.NewSpacer(),
		container.NewHBox(exitBtn),
	)

	a.window.SetContent(container.NewPadded(content))
}

// run запускает приложение
func (a *CheckApp) run() {
	a.createUI()
	a.window.ShowAndRun()
}

// newCheckApp создает новое приложение для отметки сотрудников
func newCheckApp(database *db.DB) *CheckApp {
	appInstance := app.New()
	window := appInstance.NewWindow("СКУД - Отметка сотрудников")
	window.Resize(fyne.NewSize(500, 400))

	checkApp := &CheckApp{
		app:      appInstance,
		window:   window,
		database: database,
	}

	// Обработка закрытия окна
	window.SetOnClosed(func() {
		checkApp.app.Quit()
	})

	return checkApp
}

func main() {
	// Настройка консоли для корректного отображения UTF-8 на Windows
	cmd := exec.Command("chcp", "65001")
	cmd.Run()

	// Инициализация базы данных
	database, err := db.InitDB(getExePath())
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка инициализации базы данных: %v", err), nil)
		return
	}
	defer database.Close()

	checkApp := newCheckApp(database)
	checkApp.run()
}
