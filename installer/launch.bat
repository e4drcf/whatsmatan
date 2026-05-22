@echo off
setlocal

rem Launcher created by the Inno Setup installer.
rem Resolves paths to whatsmatan.exe + bundled dependencies + downloaded model,
rem then starts the server and opens the web UI in the default browser.

set "APP_DIR=%~dp0"
set "DATA_DIR=%LOCALAPPDATA%\whatsmatan\data"

if not exist "%DATA_DIR%" mkdir "%DATA_DIR%"

set "WHISPER_CLI=%APP_DIR%whisper\whisper-cli.exe"
set "WHISPER_MODEL=%APP_DIR%ggml-large-v3.bin"
set "FFMPEG=%APP_DIR%ffmpeg\ffmpeg.exe"

if not exist "%WHISPER_CLI%"   echo Missing %WHISPER_CLI%   & pause & exit /b 1
if not exist "%WHISPER_MODEL%" echo Missing %WHISPER_MODEL% & pause & exit /b 1
if not exist "%FFMPEG%"        echo Missing %FFMPEG%        & pause & exit /b 1

start "" "http://localhost:8080"

"%APP_DIR%whatsmatan.exe" ^
  --data-dir       "%DATA_DIR%" ^
  --http-addr      :8080 ^
  --whisper-cli    "%WHISPER_CLI%" ^
  --whisper-model  "%WHISPER_MODEL%" ^
  --ffmpeg         "%FFMPEG%" ^
  --lang           uk ^
  --timeout        10m
