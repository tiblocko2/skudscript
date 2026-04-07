// Админ-панель для управления сотрудниками СКУД с графическим интерфейсом
// Позволяет добавлять, удалять и редактировать сотрудников и их карты
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"skudscript/db"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Version устанавливается через ldflags при сборке
var version = "dev"

// AdminApp представляет приложение админ-панели
type AdminApp struct {
	app      fyne.App
	window   fyne.Window
	database *db.DB
	content  *container.Scroll

	// Элементы управления
	userList    *widget.List
	statusLabel *widget.Label
}

// getExePath возвращает путь к текущему исполняемому файлу
func getExePath() string {
	exePath, err := os.Executable()
	if err != nil {
		exePath, _ = os.Getwd()
	}
	return exePath
}

// showStatus отображает сообщение о статусе операции
func (a *AdminApp) showStatus(message string, isError bool) {
	a.statusLabel.SetText(message)
	if isError {
		a.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	}
	a.statusLabel.Refresh()
}

// clearStatus очищает сообщение о статусе
func (a *AdminApp) clearStatus() {
	a.statusLabel.SetText("")
}

// showAddDialog показывает диалог добавления сотрудника
func (a *AdminApp) showAddDialog() {
	// Очищаем поля
	a.clearStatus()

	// Создаем форму
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Иванов Иван Иванович")

	cardEntry := widget.NewEntry()
	cardEntry.SetPlaceHolder("Приложите карту или введите ID")

	instruction := widget.NewLabel("📌 Приложите карту к считывателю или введите ID вручную")
	instruction.TextStyle = fyne.TextStyle{Italic: true}

	form := widget.NewForm(
		widget.NewFormItem("ФИО сотрудника", nameEntry),
		widget.NewFormItem("ID карты", cardEntry),
	)

	content := container.NewVBox(
		form,
		instruction,
	)

	// Обработчик кнопки "Создать"
	createBtn := widget.NewButton("Создать", func() {
		fullName := strings.TrimSpace(nameEntry.Text)
		cardID := strings.TrimSpace(cardEntry.Text)

		if fullName == "" {
			a.showStatus("❌ Введите ФИО сотрудника", true)
			return
		}

		if cardID == "" {
			a.showStatus("❌ Введите или приложите карту", true)
			return
		}

		// Проверяем, не используется ли уже такая карта
		exists, _, _ := a.database.CheckCardExists(cardID)
		if exists {
			a.showStatus(fmt.Sprintf("❌ Карта '%s' уже зарегистрирована", cardID), true)
			return
		}

		// Добавляем в базу данных
		err := a.database.AddUser(cardID, fullName)
		if err != nil {
			a.showStatus(fmt.Sprintf("❌ Ошибка: %v", err), true)
			return
		}

		a.showStatus("✅ Сотрудник успешно добавлен!", false)
		a.refreshUserList()

		dialog.ShowInformation("Успешно", "Сотрудник добавлен", a.window)
	})

	// Создаем свою кастомную панель с кнопками
	btnContainer := container.NewHBox(
		layout.NewSpacer(),
		widget.NewButton("Отмена", func() {
		}),
		createBtn,
	)

	finalContent := container.NewBorder(nil, btnContainer, nil, nil, container.NewPadded(content))

	d := dialog.NewCustom("Добавить сотрудника", "Отмена", finalContent, a.window)
	d.Show()
}

// showEditDialog показывает диалог редактирования сотрудника
func (a *AdminApp) showEditDialog(userID int) {
	user, err := a.database.GetUserByID(userID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка поиска сотрудника: %v", err), a.window)
		return
	}

	if user == nil {
		dialog.ShowError(fmt.Errorf("сотрудник не найден"), a.window)
		return
	}

	cardEntry := widget.NewEntry()
	cardEntry.SetText(user.CardID)

	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Сотрудник: %s", user.FullName)),
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("ID карты", cardEntry),
		),
		widget.NewLabel("📌 Приложите новую карту или введите ID вручную"),
	)

	updateBtn := widget.NewButton("Обновить", func() {
		newCardID := strings.TrimSpace(cardEntry.Text)

		if newCardID == "" {
			a.showStatus("❌ Введите или приложите карту", true)
			return
		}

		// Проверяем, не используется ли уже такая карта
		exists, existingID, _ := a.database.CheckCardExists(newCardID)
		if exists && existingID != userID {
			a.showStatus(fmt.Sprintf("❌ Карта '%s' уже зарегистрирована", newCardID), true)
			return
		}

		err := a.database.UpdateUserCard(userID, newCardID)
		if err != nil {
			a.showStatus(fmt.Sprintf("❌ Ошибка: %v", err), true)
			return
		}

		a.showStatus("✅ Карта обновлена!", false)
		a.refreshUserList()

		dialog.ShowInformation("Успешно", fmt.Sprintf("Карта сотрудника %s обновлена", user.FullName), a.window)
	})

	btnContainer := container.NewHBox(
		layout.NewSpacer(),
		widget.NewButton("Отмена", func() {
		}),
		updateBtn,
	)

	finalContent := container.NewBorder(nil, btnContainer, nil, nil, container.NewPadded(content))

	d := dialog.NewCustom("Редактировать сотрудника", "Отмена", finalContent, a.window)
	d.Show()
}

// confirmDeleteEmployee показывает диалог подтверждения удаления
func (a *AdminApp) confirmDeleteEmployee(userID int, fullName string) {
	dialog.ShowConfirm(
		"Удаление сотрудника",
		fmt.Sprintf("Вы уверены, что хотите удалить сотрудника\n\"%s\"?", fullName),
		func(confirmed bool) {
			if confirmed {
				a.deleteEmployee(userID, fullName)
			}
		},
		a.window,
	)
}

// deleteEmployee удаляет сотрудника из базы данных
func (a *AdminApp) deleteEmployee(userID int, fullName string) {
	err := a.database.DeleteUser(userID)
	if err != nil {
		a.showStatus(fmt.Sprintf("❌ Ошибка: %v", err), true)
		return
	}

	a.showStatus(fmt.Sprintf("✅ Сотрудник \"%s\" удален", fullName), false)
	a.refreshUserList()
}

// refreshUserList обновляет список сотрудников
func (a *AdminApp) refreshUserList() {
	a.userList.Refresh()
}

// createUserList создает список сотрудников
func (a *AdminApp) createUserList() *widget.List {
	users, _ := a.database.GetAllUsers()

	list := widget.NewList(
		func() int {
			return len(users)
		},
		func() fyne.CanvasObject {
			// Создаем более компактный элемент списка
			nameLabel := widget.NewLabel("ФИО")
			nameLabel.TextStyle = fyne.TextStyle{Bold: true}

			cardLabel := widget.NewLabel("Карта: XXXX")
			cardLabel.TextStyle = fyne.TextStyle{Italic: true}

			editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {})
			editBtn.Importance = widget.LowImportance

			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
			deleteBtn.Importance = widget.DangerImportance

			return container.NewHBox(
				container.NewVBox(nameLabel, cardLabel),
				layout.NewSpacer(),
				editBtn,
				deleteBtn,
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(users) {
				return
			}
			user := users[id]

			container := item.(*fyne.Container)
			vbox := container.Objects[0].(*fyne.Container)
			nameLabel := vbox.Objects[0].(*widget.Label)
			cardLabel := vbox.Objects[1].(*widget.Label)
			editBtn := container.Objects[2].(*widget.Button)
			deleteBtn := container.Objects[3].(*widget.Button)

			nameLabel.SetText(user.FullName)
			cardLabel.SetText(fmt.Sprintf("Карта: %s", user.CardID))

			editBtn.OnTapped = func() {
				a.showEditDialog(user.ID)
			}

			deleteBtn.OnTapped = func() {
				a.confirmDeleteEmployee(user.ID, user.FullName)
			}
		},
	)

	return list
}

// createMainMenu создает главное меню
func (a *AdminApp) createMainMenu() *fyne.Container {
	addBtn := widget.NewButtonWithIcon("Добавить сотрудника", theme.ContentAddIcon(), func() {
		a.showAddDialog()
	})
	addBtn.Importance = widget.HighImportance

	refreshBtn := widget.NewButtonWithIcon("Обновить список", theme.ViewRefreshIcon(), func() {
		a.refreshUserList()
	})

	quitBtn := widget.NewButtonWithIcon("Выход", theme.CancelIcon(), func() {
		a.app.Quit()
	})

	buttons := container.NewHBox(addBtn, refreshBtn, layout.NewSpacer(), quitBtn)

	return container.NewBorder(buttons, nil, nil, nil, buttons)
}

// createUI создает пользовательский интерфейс
func (a *AdminApp) createUI() {
	a.statusLabel = widget.NewLabel("")
	a.userList = a.createUserList()

	mainMenu := a.createMainMenu()

	// Заголовок списка
	listLabel := widget.NewLabel("Список сотрудников:")
	listLabel.TextStyle = fyne.TextStyle{Bold: true}

	content := container.NewVBox(
		listLabel,
		a.userList,
	)

	a.content = container.NewScroll(content)
	a.content.SetMinSize(fyne.NewSize(0, 400))

	mainContent := container.NewBorder(mainMenu, a.statusLabel, nil, nil, a.content)
	a.window.SetContent(mainContent)
}

// run запускает приложение
func (a *AdminApp) run() {
	a.createUI()
	a.window.ShowAndRun()
}

// newAdminApp создает новое приложение админ-панели
func newAdminApp(database *db.DB) *AdminApp {
	appInstance := app.New()
	window := appInstance.NewWindow("СКУД - Админ-панель")
	window.Resize(fyne.NewSize(700, 600))

	return &AdminApp{
		app:      appInstance,
		window:   window,
		database: database,
	}
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

	adminApp := newAdminApp(database)
	adminApp.run()
}
