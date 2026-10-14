@echo off
rem Builds the API into the project root and runs it. Extra arguments are
rem passed to the executable. The .env in the project root is loaded on startup.
setlocal

cd /d "%~dp0"

go build -C .\cmd\api -o ..\..\email-tracker.exe
if errorlevel 1 exit /b 1

.\email-tracker.exe %*
exit /b %errorlevel%
