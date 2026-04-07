@echo off
chcp 65001 >nul
echo ============================================
echo Создание релиза для GitHub
echo ============================================
echo.

cd /d "%~dp0"

REM Проверяем git
where git >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ОШИБКА] Git не найден.
    pause
    exit /b 1
)

REM Получаем текущую дату для версии
for /f "tokens=2-4 delims=/ " %%a in ('echo %date%') do (
    set DAY=%%a
    set MONTH=%%b
    set YEAR=%%c
)
set DEFAULT_TAG=v%YEAR%.%MONTH%.%DAY%

REM Спрашиваем версию
set /p VERSION="Введите версию (по умолчанию %DEFAULT_TAG%): "
if "%VERSION%"=="" set VERSION=%DEFAULT_TAG%

echo.
echo [INFO] Версия: %VERSION%
echo.

REM Проверяем, существует ли уже такой тег
git show-ref --tags | findstr "refs/tags/%VERSION%" >nul 2>nul
if %ERRORLEVEL% equ 0 (
    echo [ОШИБКА] Тег %VERSION% уже существует!
    echo Выберите другую версию или удалите старый тег:
    echo   git tag -d %VERSION%
    pause
    exit /b 1
)

REM Сначала собираем проект
echo [1/4] Сборка проекта...
call build.bat
if %ERRORLEVEL% neq 0 (
    echo [ОШИБКА] Не удалось собрать проект
    pause
    exit /b 1
)

echo.
echo [2/4] Создание ZIP архива...
call create-archive.bat
if %ERRORLEVEL% neq 0 (
    echo [ОШИБКА] Не удалось создать архив
    pause
    exit /b 1
)

echo.
echo [3/4] Создание git тега...
git tag -a %VERSION% -m "Release %VERSION%"
if %ERRORLEVEL% neq 0 (
    echo [ОШИБКА] Не удалось создать тег
    pause
    exit /b 1
)
echo [OK] Тег %VERSION% создан
echo.

echo [4/4] Инструкции по публикации релиза
echo.
echo ============================================
echo ГОТОВО! Теперь выполните команды вручную:
echo ============================================
echo.
echo 1. Отправьте тег на GitHub:
echo    git push origin %VERSION%
echo.
echo 2. Перейдите на GitHub:
echo    https://github.com/your-org/skudscript/releases
echo.
echo 3. Нажмите "Draft a new release" или отредактируйте тег %VERSION%
echo.
echo 4. Прикрепите файлы из папки dist\:
echo    - admin-app.exe
echo    - check-app.exe
echo    - skudscript_release.zip (полный архив)
echo.
echo 5. Добавьте описание релиза и нажмите "Publish release"
echo.
echo ============================================
echo АВТОМАТИЧЕСКАЯ АЛЬТЕРНАТИВА (gh CLI):
echo ============================================
echo.
echo Если у вас установлен GitHub CLI (gh), выполните:
echo.
echo gh release create %VERSION% ^
echo   --title "Release %VERSION%" ^
echo   --notes "Автоматический релиз %VERSION%" ^
echo   dist/admin-app.exe ^
echo   dist/check-app.exe ^
echo   skudscript_release.zip
echo.
pause
