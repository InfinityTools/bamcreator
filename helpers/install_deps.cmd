@echo off

REM *** This script makes sure specified packages are installed locally. ***
REM *** Syntax: install_deps [-u] package1 [package2 [...]]              ***
REM *** Option -u: Update installed packages                             ***

REM Checking Go compiler
where /q go || (
  echo Go compiler not found.
  goto Failed
)

REM Processing packages
set update=0
:ArgsLoop
if "%~1"=="" goto Success

REM Enable force package update
if "%~1"=="-u" (
  set update=1
  goto ArgsUpdate
)

REM installing package
call :InstallPackage %update% %~1 || goto Failed

:ArgsUpdate
shift
goto ArgsLoop


:Failed
exit /b 1

:Success
exit /b 0

REM Function syntax: InstallPackage update_flag package_name
:InstallPackage
if "%~2"=="" exit /b 0
go list %~2 >nul 2>&1 && (
  REM Don't update existing packages?
  if /i %~1 EQU 0 exit /b 0
)
echo Installing package %~2
go get -f -u %~2 || exit /b 1
exit /b 0
