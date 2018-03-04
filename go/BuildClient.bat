@echo off
SETLOCAL
IF EXIST "..\Client.exe" (
	del .\builds\Client.exe
	echo Removing Previous Client
)
go build -i -o ..\Client.exe client.go > buildInfo
IF EXIST "..\Client.exe" (
	echo Build Successful: Client.exe
	del buildInfo
) ELSE (
	type buildInfo
)

echo Build Complete
