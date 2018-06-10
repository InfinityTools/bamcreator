@echo off

setlocal
set builddir=%~dp0
set builddir=%builddir:~0,-1%

REM Checking Go compiler
where /q go || (
  echo Go compiler not found.
  goto Failed
)

set pkgRoot=github.com/InfinityTools

set pkgPrefix=%pkgRoot%/bamcreator
set pkgBinRoot=bin
set pkgBinExt=.exe

REM Dependencies
set pkgCharmap=golang.org/x/text/encoding/charmap
set pkgBmp=golang.org/x/image/bmp
set pkgThreadpool=github.com/pbenner/threadpool
set pkgBinpack=%pkgRoot%/go-binpack2d
set pkgCmdArgs=%pkgRoot%/go-cmdargs
set pkgIETools=%pkgRoot%/go-ietools
set pkgIEToolsBuffers=%pkgRoot%/go-ietools/buffers
set pkgIEToolsPvrz=%pkgRoot%/go-ietools/pvrz
set pkgImagequant=%pkgRoot%/go-imagequant
set pkgLogging=%pkgRoot%/go-logging
set pkgSquish=%pkgRoot%/go-squish

REM Supported targets
set targetBamCreator=bamcreator
set targetBamConv=bamconv
set targetBamGen=bamgen

REM Initializing with default target
set target=%targetBamCreator%

REM Evaluating command line parameters
set bin_flags=-ldflags -s
set skipdeps=0
set compress=0
:ArgsLoop
if "%~1"=="" goto ArgsFinished

if "%~1"=="bamcreator" (
  set target=bamcreator
  goto ArgsUpdate
)

if "%~1"=="bamconv" (
  set target=bamconv
  goto ArgsUpdate
)

if "%~1"=="bamgen" (
  set target=bamgen
  goto ArgsUpdate
)

if "%~1"=="--debug" (
  set bin_flags=
  goto ArgsUpdate
)

if "%~1"=="--nodeps" (
  set skipdeps=1
  goto ArgsUpdate
)

if "%~1"=="--compress" (
  set compress=1
  goto ArgsUpdate
)

if "%~1"=="--update" (
  set get_flags=-u
  set build_flags=-a
  goto ArgsUpdate
)

if "%~1"=="--help" (
echo Usage: %~n0%~x0 [options] [target]
echo.
echo Options:
echo   --debug          Don't strip debugging symbols
echo   --nodeps         Don't check dependencies
echo   --update         Force updating dependencies
echo   --compress       Compress the binary with upx if available
echo   --help           This help
echo.
echo Available targets:
echo   bamcreator [default]
echo.
call :DetectSystem
echo The resulting binary is placed into the folder "bin/%libos%/%libarch%".
goto FinishedNoMessage
)

:ArgsUpdate
shift
goto ArgsLoop

:ArgsFinished
if defined bin_flags (
  echo Building %target% release version
) else (
  echo Building %target% debug version
)

if /i %skipdeps% NEQ 0 goto SkipDependencies
REM Iterating over list of dependencies: simple check and install-on-demand
for %%a in (%pkgCharmap%
            %pkgBmp%
            %pkgThreadpool%
            %pkgBinpack%
            %pkgCmdArgs%
            %pkgIETools%
            %pkgIEToolsBuffers%
            %pkgIEToolsPvrz%
            %pkgImagequant%
            %pkgLogging%
            %pkgSquish%) do (
  echo Checking %%a ...
  call "%builddir%\helpers\install_deps.cmd" %get_flags% %%a || goto Failed
)

:SkipDependencies
call :DetectSystem
echo Detected: os=%libos%, arch=%libarch%

REM Use static linking on Windows if possible
if "%libos%"=="windows" (
  set CGO_LDFLAGS=-static -static-libstdc++ %CGO_LDFLAGS%
)

REM Starting build operation
set pkgBinPath=%builddir%/%pkgBinRoot%/%libos%/%libarch%/%target%%pkgBinExt%
echo Building "%pkgBinPath%"...
REM echo go build -o "%pkgBinPath%" %build_flags% %bin_flags% %pkgPrefix%/%target%
go build -o "%pkgBinPath%" %build_flags% %bin_flags% %pkgPrefix%/%target% && goto Compress || goto Failed


:Compress
REM Applying compression if needed
if /i %compress% NEQ 1 goto Finished
if not exist "%pkgBinPath%" goto Finished
where /q upx || (
  echo Could not find upx. Skipping compression....
  goto Finished
)
echo Compressing binary. This may take a while...
upx --best -q "%pkgBinPath%" >nul || (
  echo Compression failed.
  goto FailedNoMessage
)

:Finished
echo Finished.
:FinishedNoMessage
endlocal
exit /b 0

:Failed
echo Cancelled.
:FailedNoMessage
endlocal
exit /b 1


REM Autodetect system (use as function)
:DetectSystem
for /f "tokens=* usebackq" %%a in (`go env GOOS`) do (
  set libos=%%a
)
for /f "tokens=* usebackq" %%a in (`go env GOARCH`) do (
  set libarch=%%a
)
exit /b 0
