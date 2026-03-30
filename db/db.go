// Пакет db предоставляет функции для работы с базой данных SQLite
package db

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// User представляет сотрудника в базе данных
type User struct {
	ID       int
	CardID   string
	FullName string
}

// DB содержит подключение к базе данных
type DB struct {
	conn *sql.DB
}

// InitDB инициализирует базу данных и создает таблицу users, если она не существует
// pathToExe - путь к исполняемому файлу для определения расположения БД
func InitDB(pathToExe string) (*DB, error) {
	// База данных хранится в той же папке, что и exe файл
	dbPath := filepath.Join(filepath.Dir(pathToExe), "skudscript.db")

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия базы данных: %w", err)
	}

	// Создаем таблицу users, если она не существует
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		card_id TEXT UNIQUE NOT NULL,
		full_name TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_card_id ON users(card_id);
	`

	_, err = conn.Exec(createTableSQL)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close закрывает подключение к базе данных
func (d *DB) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// AddUser добавляет нового сотрудника в базу данных
func (d *DB) AddUser(cardID, fullName string) error {
	insertSQL := `INSERT INTO users (card_id, full_name) VALUES (?, ?)`
	result, err := d.conn.Exec(insertSQL, cardID, fullName)
	if err != nil {
		return fmt.Errorf("ошибка добавления сотрудника: %w", err)
	}

	// Проверяем, была ли затронута хотя бы одна строка
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return fmt.Errorf("не удалось добавить сотрудника")
	}

	return nil
}

// UpdateUserCard обновляет карту сотрудника по ID
func (d *DB) UpdateUserCard(userID int, newCardID string) error {
	updateSQL := `UPDATE users SET card_id = ? WHERE id = ?`
	result, err := d.conn.Exec(updateSQL, newCardID, userID)
	if err != nil {
		return fmt.Errorf("ошибка обновления карты: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return fmt.Errorf("сотрудник не найден")
	}

	return nil
}

// UpdateUserCardByCardID обновляет карту сотрудника по текущему card_id
func (d *DB) UpdateUserCardByCardID(oldCardID, newCardID string) error {
	updateSQL := `UPDATE users SET card_id = ? WHERE card_id = ?`
	result, err := d.conn.Exec(updateSQL, newCardID, oldCardID)
	if err != nil {
		return fmt.Errorf("ошибка обновления карты: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return fmt.Errorf("сотрудник не найден")
	}

	return nil
}

// DeleteUser удаляет сотрудника по ID
func (d *DB) DeleteUser(userID int) error {
	deleteSQL := `DELETE FROM users WHERE id = ?`
	result, err := d.conn.Exec(deleteSQL, userID)
	if err != nil {
		return fmt.Errorf("ошибка удаления сотрудника: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return fmt.Errorf("сотрудник не найден")
	}

	return nil
}

// GetUserByCardID находит сотрудника по ID карты
func (d *DB) GetUserByCardID(cardID string) (*User, error) {
	querySQL := `SELECT id, card_id, full_name FROM users WHERE card_id = ?`
	row := d.conn.QueryRow(querySQL, cardID)

	var user User
	err := row.Scan(&user.ID, &user.CardID, &user.FullName)
	if err == sql.ErrNoRows {
		return nil, nil // Сотрудник не найден
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска сотрудника: %w", err)
	}

	return &user, nil
}

// GetUserByID находит сотрудника по ID
func (d *DB) GetUserByID(userID int) (*User, error) {
	querySQL := `SELECT id, card_id, full_name FROM users WHERE id = ?`
	row := d.conn.QueryRow(querySQL, userID)

	var user User
	err := row.Scan(&user.ID, &user.CardID, &user.FullName)
	if err == sql.ErrNoRows {
		return nil, nil // Сотрудник не найден
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска сотрудника: %w", err)
	}

	return &user, nil
}

// GetAllUsers возвращает список всех сотрудников
func (d *DB) GetAllUsers() ([]User, error) {
	querySQL := `SELECT id, card_id, full_name FROM users ORDER BY full_name`
	rows, err := d.conn.Query(querySQL)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка сотрудников: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.CardID, &user.FullName)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения данных: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// CheckCardExists проверяет, существует ли карта с таким ID
func (d *DB) CheckCardExists(cardID string) (bool, int, error) {
	querySQL := `SELECT id FROM users WHERE card_id = ?`
	row := d.conn.QueryRow(querySQL, cardID)

	var userID int
	err := row.Scan(&userID)
	if err == sql.ErrNoRows {
		return false, 0, nil // Карта не найдена
	}
	if err != nil {
		return false, 0, fmt.Errorf("ошибка проверки карты: %w", err)
	}

	return true, userID, nil
}
