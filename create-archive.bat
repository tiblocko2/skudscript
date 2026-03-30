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

echo [OK] Файлы скопированы
echo.

REM Проверяем наличие PowerShell для создания ZIP
powershell -Command "Compress-Archive -Path '%ARCHIVE_DIR%\*' -DestinationPath 'skudscript_release.zip' -Force" 2>nul
if %ERRORLEVEL% equ 0 (
    echo [OK] Архив skudscript_release.zip создан
    echo.
    echo Для использования:
    echo   1. Скопируйте skudscript_release.zip в нужную папку
    echo   2. Извлеките содержимое архива
    echo   3. Запускайте admin-app.exe или check-app.exe
    echo.
    echo Содержимое архива:
    echo   - admin-app.exe  (админ-панель с GUI)
    echo   - check-app.exe  (отметка сотрудников с GUI)
    echo   - README.txt     (инструкция)
) else (
    echo [ПРЕДУПРЕЖДЕНИЕ] Не удалось создать ZIP архив
    echo   Файлы скопированы в папку: %ARCHIVE_DIR%
    echo   Создайте архив вручную
)

echo.
echo ============================================
echo Готово!
echo ============================================
echo.
pause
