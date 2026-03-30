// Админ-панель для управления сотрудниками системы СКУД
// Позволяет добавлять, удалять и редактировать сотрудников и их карты
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"skudscript/db"
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

// showMenu отображает главное меню программы
func showMenu() {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🔧 СИСТЕМА УПРАВЛЕНИЯ СОТРУДНИКАМИ СКУД")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("1. ➕ Добавить сотрудника")
	fmt.Println("2. 📋 Посмотреть список сотрудников")
	fmt.Println("3. 🚪 Выход")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Print("\nВыберите действие (1-3): ")
}

// showEmployeeMenu отображает меню работы со списком сотрудников
func showEmployeeMenu() {
	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Println("📋 СПИСОК СОТРУДНИКОВ")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("0. ↩️ Назад в главное меню")
	fmt.Println("   Введите ID сотрудника для редактирования")
	fmt.Println(strings.Repeat("-", 50))
}

// showEditMenu отображает меню редактирования сотрудника
func showEditMenu(user *db.User) {
	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Printf("👤 Сотрудник: %s\n", user.FullName)
	fmt.Printf("🔑 Текущая карта: %s\n", user.CardID)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("1. 🔄 Изменить карту")
	fmt.Println("2. 🗑️ Удалить сотрудника")
	fmt.Println("0. ↩️ Назад")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Print("\nВыберите действие (0-2): ")
}

// addEmployee добавляет нового сотрудника
func addEmployee(database *db.DB, reader *bufio.Reader) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("➕ ДОБАВЛЕНИЕ СОТРУДНИКА")
	fmt.Println(strings.Repeat("=", 50))

	// Ввод ФИО сотрудника
	fmt.Print("\nВведите ФИО сотрудника: ")
	fullName, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("❌ Ошибка ввода ФИО")
		return
	}
	fullName = strings.TrimSpace(fullName)

	if fullName == "" {
		fmt.Println("❌ ФИО не может быть пустым")
		return
	}

	// Считывание карты
	cardID, err := readCardID(reader)
	if err != nil {
		fmt.Println("❌ Ошибка считывания карты")
		return
	}

	// Проверка, не используется ли уже такая карта
	exists, _, _ := database.CheckCardExists(cardID)
	if exists {
		fmt.Printf("❌ Карта с ID '%s' уже зарегистрирована!\n", cardID)
		return
	}

	// Добавление в базу данных
	err = database.AddUser(cardID, fullName)
	if err != nil {
		fmt.Printf("❌ Ошибка добавления: %v\n", err)
		return
	}

	fmt.Println("\n✅ Сотрудник успешно добавлен!")
	fmt.Printf("   ФИО: %s\n", fullName)
	fmt.Printf("   Карта: %s\n", cardID)
}

// editEmployee позволяет редактировать или удалить сотрудника
func editEmployee(database *db.DB, reader *bufio.Reader, userID int) {
	user, err := database.GetUserByID(userID)
	if err != nil {
		fmt.Printf("❌ Ошибка поиска сотрудника: %v\n", err)
		return
	}

	if user == nil {
		fmt.Println("❌ Сотрудник с таким ID не найден")
		return
	}

	for {
		showEditMenu(user)

		choice, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("❌ Ошибка ввода")
			return
		}
		choice = strings.TrimSpace(choice)

		switch choice {
		case "0":
			return
		case "1":
			// Изменение карты
			fmt.Println("\n🔄 ИЗМЕНЕНИЕ КАРТЫ")
			fmt.Println("Старая карта:", user.CardID)

			// Считывание новой карты
			newCardID, err := readCardID(reader)
			if err != nil {
				fmt.Println("❌ Ошибка считывания новой карты")
				return
			}

			// Проверка, не используется ли уже такая карта
			exists, _, _ := database.CheckCardExists(newCardID)
			if exists {
				fmt.Printf("❌ Карта с ID '%s' уже зарегистрирована!\n", newCardID)
				return
			}

			// Обновление карты
			err = database.UpdateUserCard(user.ID, newCardID)
			if err != nil {
				fmt.Printf("❌ Ошибка обновления: %v\n", err)
				return
			}

			fmt.Println("\n✅ Карта успешно обновлена!")
			fmt.Printf("   Новая карта: %s\n", newCardID)
			user.CardID = newCardID

		case "2":
			// Удаление сотрудника
			fmt.Print("\n⚠️ Вы уверены, что хотите удалить сотрудника? (д/н): ")
			confirm, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("❌ Ошибка подтверждения")
				return
			}
			confirm = strings.TrimSpace(strings.ToLower(confirm))

			if confirm == "д" || confirm == "y" {
				err = database.DeleteUser(user.ID)
				if err != nil {
					fmt.Printf("❌ Ошибка удаления: %v\n", err)
					return
				}
				fmt.Println("\n✅ Сотрудник успешно удален")
				return
			}
			fmt.Println("❌ Удаление отменено")

		default:
			fmt.Println("❌ Неверный выбор, попробуйте снова")
		}
	}
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

	fmt.Println("\n🚀 Запуск системы управления СКУД...")

	// Инициализация базы данных
	database, err := db.InitDB(getExePath())
	if err != nil {
		fmt.Printf("❌ Ошибка инициализации базы данных: %v\n", err)
		return
	}
	defer database.Close()

	reader := bufio.NewReader(os.Stdin)

	// Главный цикл программы
	for {
		showMenu()

		choice, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("❌ Ошибка ввода, попробуйте снова")
			continue
		}
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			// Добавить сотрудника
			addEmployee(database, reader)

		case "2":
			// Показать список сотрудников
			for {
				users, _ := database.GetAllUsers()
				showEmployeeMenu()

				if len(users) == 0 {
					fmt.Println("\n📭 Список сотрудников пуст")
					break
				}

				// Отображение таблицы сотрудников
				fmt.Printf("\n%-5s %-25s %-20s\n", "ID", "ФИО", "Карта")
				fmt.Println(strings.Repeat("-", 50))

				for _, user := range users {
					name := user.FullName
					if len(name) > 24 {
						name = name[:21] + "..."
					}
					fmt.Printf("%-5d %-25s %-20s\n", user.ID, name, user.CardID)
				}

				fmt.Print("\nВведите ID сотрудника (или 0 для выхода): ")
				input, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("❌ Ошибка ввода")
					break
				}
				input = strings.TrimSpace(input)

				if input == "0" {
					break
				}

				userID, err := strconv.Atoi(input)
				if err != nil {
					fmt.Println("❌ Неверный формат ID")
					continue
				}

				editEmployee(database, reader, userID)
			}

		case "3":
			// Выход
			fmt.Println("\n👋 До свидания!")
			return

		default:
			fmt.Println("❌ Неверный выбор, введите число от 1 до 3")
		}
	}
}
