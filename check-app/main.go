// Приложение для отметки сотрудников СКУД
// Работает в системном трее, перехватывает ввод карт на глобальном уровне
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unsafe"

	"skudscript/db"

	"fyne.io/systray"
	"github.com/xuri/excelize/v2"
	"golang.org/x/sys/windows"
)

// Version устанавливается через ldflags при сборке
var version = "dev"

// === Windows API константы ===

const (
	WH_KEYBOARD_LL = 13
	WM_KEYDOWN     = 0x0100
	WM_SYSKEYDOWN  = 0x0104
	VK_BACK        = 0x08
	VK_RETURN      = 0x0D
	VK_ESCAPE      = 0x1B
	VK_0           = 0x30
	VK_9           = 0x39
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procGetModuleHandle     = kernel32.NewProc("GetModuleHandleW")
	procMessageBox          = user32.NewProc("MessageBoxW")
	procGetMessage          = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessage     = user32.NewProc("DispatchMessageW")
)

// KBDLLHOOKSTRUCT структура для перехвата клавиш
type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type HHOOK windows.Handle

// MSG структура для цикла сообщений
type MSG struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// === Глобальные переменные ===

var (
	hook        HHOOK
	cardBuffer  strings.Builder
	keyTimes    []time.Time
	bufferMu    sync.Mutex
	lastKeyTime time.Time
	database    *db.DB
)

// isValidCardID проверяет, что строка содержит ровно 10 цифр
func isValidCardID(s string) bool {
	if len(s) != 10 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// utf16FromString преобразует строку в UTF-16 для Windows API
func utf16FromString(s string) *uint16 {
	ret, _ := syscall.UTF16PtrFromString(s)
	return ret
}

// showNotification показывает уведомление через Windows MessageBox
func showNotification(title, message string) {
	// Используем MessageBox для уведомлений
	procMessageBox.Call(
		0,
		uintptr(unsafe.Pointer(utf16FromString(message))),
		uintptr(unsafe.Pointer(utf16FromString(title))),
		0x40, // MB_ICONINFORMATION
	)
}

// processCardID обрабатывает ID карты
func processCardID(cardID string) {
	if !isValidCardID(cardID) {
		return
	}

	// Поиск сотрудника по карте
	user, err := database.GetUserByCardID(cardID)
	if err != nil {
		showNotification("Ошибка СКУД", fmt.Sprintf("Ошибка поиска: %v", err))
		return
	}

	if user == nil {
		showNotification("Карта не найдена", "Сотрудник с такой картой не зарегистрирован")
		return
	}

	// Добавляем запись в отчет
	currentDate := time.Now()
	err = addRecordToReport(getExePath(), currentDate, user.CardID, user.FullName)
	if err != nil {
		showNotification("Ошибка отчета", fmt.Sprintf("Ошибка: %v", err))
		return
	}

	// Успешная отметка
	currentTime := time.Now().Format("15:04:05")
	currentDateStr := currentDate.Format("02.01.2006")
	showNotification(
		"✅ Отметка успешна!",
		fmt.Sprintf("%s\n%s %s", user.FullName, currentDateStr, currentTime),
	)
}

// keyboardHook процедура перехвата клавиш
func keyboardHook(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 && (wParam == WM_KEYDOWN || wParam == WM_SYSKEYDOWN) {
		kbdStruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))

		// Проверяем специальные клавиши
		if kbdStruct.VkCode == VK_BACK || kbdStruct.VkCode == VK_ESCAPE {
			bufferMu.Lock()
			cardBuffer.Reset()
			keyTimes = keyTimes[:0]
			bufferMu.Unlock()
			return callNextHook(nCode, wParam, lParam)
		}

		// Проверяем Enter - обработка карты
		if kbdStruct.VkCode == VK_RETURN {
			bufferMu.Lock()
			cardID := cardBuffer.String()
			if cardID != "" {
				go processCardID(cardID)
			}
			cardBuffer.Reset()
			keyTimes = keyTimes[:0]
			bufferMu.Unlock()
			return callNextHook(nCode, wParam, lParam)
		}

		// Обработка цифр (0-9)
		now := time.Now()
		bufferMu.Lock()

		// ЗАКОММЕНТИРОВАНО для тестирования без считывателя:
		// Если прошло больше 100 мс с последнего символа - сбрасываем буфер
		// if !lastKeyTime.IsZero() && now.Sub(lastKeyTime) > 100*time.Millisecond {
		// 	cardBuffer.Reset()
		// 	keyTimes = keyTimes[:0]
		// }

		lastKeyTime = now

		// Проверяем, что введена цифра (VK 0-9 = 48-57)
		if kbdStruct.VkCode >= VK_0 && kbdStruct.VkCode <= VK_9 {
			cardBuffer.WriteByte(byte('0' + (kbdStruct.VkCode - VK_0)))
			keyTimes = append(keyTimes, now)
		} else {
			// Нецифровой символ - сбрасываем
			cardBuffer.Reset()
			keyTimes = keyTimes[:0]
		}

		bufferMu.Unlock()
	}

	return callNextHook(nCode, wParam, lParam)
}

func callNextHook(nCode int, wParam, lParam uintptr) uintptr {
	ret, _, _ := procCallNextHookEx.Call(
		uintptr(hook),
		uintptr(nCode),
		wParam,
		lParam,
	)
	return ret
}

// messageLoop цикл обработки сообщений Windows для работы глобального хука
func messageLoop() {
	var msg MSG
	for {
		ret, _, _ := procGetMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)
		if ret == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// hookReady канал для сигнализации о готовности хука
var hookReady = make(chan struct{})

// setHook устанавливает глобальный хук клавиатуры
func setHook() error {
	// Запускаем хук в отдельной goroutine с собственным циклом сообщений
	go func() {
		hModule, _, _ := procGetModuleHandle.Call(0)

		h, _, err := procSetWindowsHookEx.Call(
			WH_KEYBOARD_LL,
			syscall.NewCallback(keyboardHook),
			hModule,
			0, // 0 = глобальный хук
		)

		if h == 0 {
			fmt.Fprintf(os.Stderr, "Ошибка установки хука: %v\n", err)
			close(hookReady)
			return
		}

		hook = HHOOK(h)
		close(hookReady)

		// Запускаем цикл обработки сообщений — это блокирует goroutine
		messageLoop()
	}()

	// Ждем инициализации хука
	<-hookReady

	if hook == 0 {
		return fmt.Errorf("не удалось установить глобальный хук клавиатуры")
	}

	return nil
}

// unhook удаляет хук
func unhook() {
	if hook != 0 {
		procUnhookWindowsHookEx.Call(uintptr(hook))
		hook = 0
	}
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

// ensureReportDir создает папку для отчетов
func ensureReportDir(exePath string) error {
	reportDir := getReportDir(exePath)
	return os.MkdirAll(reportDir, 0755)
}

// createNewReport создает новый Excel файл отчета
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

// checkReportExists проверяет существование файла отчета
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

// onReady вызывается когда трей готов
func onReady(dbPath string) {
	var err error
	database, err = db.InitDB(dbPath)
	if err != nil {
		showNotification("Ошибка СКУД", fmt.Sprintf("Ошибка БД: %v", err))
		systray.Quit()
		return
	}

	// Устанавливаем хук клавиатуры
	err = setHook()
	if err != nil {
		showNotification("Ошибка СКУД", fmt.Sprintf("Не удалось перехватить клавиатуру: %v", err))
		systray.Quit()
		return
	}

	systray.SetTitle("СКУД")
	systray.SetTooltip("СКУД - Считыватель карт активен")

	// Создаем меню
	mStatus := systray.AddMenuItem("🟢 СКУД активен", "Приложение работает в фоне")
	mStatus.Disable()
	systray.AddSeparator()
	mTest := systray.AddMenuItem("🧪 Тест уведомления", "Проверить уведомления")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("❌ Выход", "Закрыть приложение")

	// Обработчики меню
	go func() {
		for {
			select {
			case <-mTest.ClickedCh:
				showNotification("Тест СКУД", "Уведомления работают корректно\nПриложение готово к приёму карт")
			case <-mQuit.ClickedCh:
				unhook()
				database.Close()
				systray.Quit()
			}
		}
	}()
}

// onExit вызывается при выходе
func onExit() {
	unhook()
	if database != nil {
		database.Close()
	}
}

func main() {
	// Инициализация БД
	dbPath := getExePath()

	// Запускаем systray
	systray.Run(func() {
		onReady(dbPath)
	}, onExit)
}
