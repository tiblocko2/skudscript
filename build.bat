@echo off
chcp 65001 >nul
echo ============================================
echo Сборка приложений СКУД в EXE файлы (GUI)
echo ============================================
echo.

REM Переходим в корневую директорию проекта
cd /d "%~dp0"

REM Проверяем наличие Go
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ОШИБКА] Go не найден. Установите Go с https://golang.org/dl/
    pause
    exit /b 1
)

echo [INFO] Версия Go:
go version
echo.

REM Создаем папку для готовых файлов
if not exist "dist" mkdir dist

REM Сборка админ-панели
echo [1/2] Сборка admin-app.exe...
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
go build -o dist\admin-app.exe -ldflags="-s -w -H windowsgui" admin-app\main.go
if %ERRORLEVEL% neq 0 (
    echo [ОШИБКА] Не удалось собрать admin-app.exe
    pause
    exit /b 1
)
echo [OK] admin-app.exe создан
echo.

REM Сборка приложения отметки
echo [2/2] Сборка check-app.exe...
go build -o dist\check-app.exe -ldflags="-s -w -H windowsgui" check-app\main.go
if %ERRORLEVEL% neq 0 (
    echo [ОШИБКА] Не удалось собрать check-app.exe
    pause
    exit /b 1
)
echo [OK] check-app.exe создан
echo.

echo ============================================
echo Сборка завершена успешно!
echo ============================================
echo.
echo Готовые файлы находятся в папке: dist\
echo   - admin-app.exe  (админ-панель с GUI)
echo   - check-app.exe  (отметка сотрудников с GUI)
echo.
echo Для создания архива ZIP запустите: create-archive.bat
echo.
pause
