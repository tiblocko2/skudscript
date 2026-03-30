@echo off
chcp 65001 >nul
echo ============================================
echo Создание ZIP архива с приложениями СКУД
echo ============================================
echo.

cd /d "%~dp0"

REM Проверяем, существует ли папка dist
if not exist "dist" (
    echo [ОШИБКА] Папка dist не найдена. Сначала запустите build.bat
    pause
    exit /b 1
)

REM Создаем папку для архива
set ARCHIVE_DIR=skudscript_release
if exist "%ARCHIVE_DIR%" rmdir /s /q "%ARCHIVE_DIR%"
mkdir "%ARCHIVE_DIR%"

REM Копируем exe файлы
echo [INFO] Копирование файлов...
copy dist\admin-app.exe "%ARCHIVE_DIR%\" >nul
copy dist\check-app.exe "%ARCHIVE_DIR%\" >nul
copy README.txt "%ARCHIVE_DIR%\" >nul

REM Создаем файл инструкции
echo [INFO] Создание инструкции...
(
echo ============================================
echo СИСТЕМА УПРАВЛЕНИЯ СКУД - ИНСТРУКЦИЯ
echo ============================================
echo.
echo ПАПКИ И ФАЙЛЫ:
echo   Otchet\        - Папка для Excel отчетов (создается автоматически)
echo   skudscript.db  - База данных SQLite (создается автоматически)
echo.
echo ПРИЛОЖЕНИЯ:
echo.
echo 1. admin-app.exe - Админ-панель для управления сотрудниками
echo    Запуск: Дважды кликните по файлу или через командную строку
echo.
echo    ФУНКЦИИ:
echo    1. Добавить сотрудника
echo       - Введите ФИО сотрудника
echo       - Приложите карту к считывателю
echo       - Сотрудник будет добавлен в базу
echo.
echo    2. Посмотреть список сотрудников
echo       - Отображается таблица всех сотрудников
echo       - Введите ID сотрудника для редактирования
echo       - 0 - выход в главное меню
echo.
echo    3. Выход
echo       - Закрытие программы
echo.
echo    ПРИ РЕДАКТИРОВАНИИ:
echo    1. Изменить карту - приложите новую карту
echo    2. Удалить сотрудника - подтвердите удаление
echo    0. Назад в меню
echo.
echo.
echo 2. check-app.exe - Приложение для отметки сотрудников
echo    Запуск: Дважды кликните по файлу или через командную строку
echo.
echo    ИСПОЛЬЗОВАНИЕ:
echo    1. Приложите карту к считывателю
echo    2. Программа автоматически запишет отметку в Excel
echo    3. Нажмите Enter для выхода
echo.
echo    ОТЧЕТЫ:
echo    - Отчеты хранятся в папке Otchet\
echo    - Имя файла: дата в формате ГГГГ-ММ-ДД.xlsx
echo    - Столбцы: Время, ФИО, Карточка
echo    - Новый отчет создается при первой отметке дня
echo.
echo ============================================
echo ТЕХНИЧЕСКАЯ ИНФОРМАЦИЯ
echo ============================================
echo.
echo Требования:
echo   - Windows 7 и выше
echo   - Считыватель карт (работает как клавиатура)
echo.
echo Примечания:
echo   - База данных создается автоматически при первом запуске
echo   - Все файлы хранятся в одной папке с exe файлами
echo   - Для переноса скопируйте всю папку с файлами
echo.
echo ============================================
) > "%ARCHIVE_DIR%\README.txt"

echo [OK] Файлы скопированы
echo.

REM Проверяем наличие PowerShell для создания ZIP
powershell -Command "Compress-Archive -Path '%ARCHIVE_DIR%\*' -DestinationPath 'skudscript_release.zip' -Force" 2>nul
if %ERRORLEVEL% equ 0 (
    echo [OK] Архив skudscript_release.zip создан
    echo.
    echo Для извлечения:
    echo   1. Скопируйте skudscript_release.zip в нужную папку
    echo   2. Извлеките содержимое архива
    echo   3. Запускайте admin-app.exe или check-app.exe
) else (
    echo [ПРЕДУПРЕЖДЕНИЕ] Не удалось создать ZIP архив
    echo   Установите PowerShell или создайте архив вручную
)

echo.
echo ============================================
echo Готово!
echo ============================================
echo.
pause
