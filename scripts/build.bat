@echo off

set GOARCH=386

set /p ip="Specify default IP address for client (or leave empty): "

if defined ip (
    go build -ldflags "-X main.defaultClientAddr=%ip%" -o ../build/multispeaker.exe ../
) else (
    go build -o ../build/multispeaker.exe ../
)
