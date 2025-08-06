go build -o builds/win10_amd64/gocl.exe main.go

SET GOOS=linux&&SET GOARCH=amd64&&go build -o builds/linux_amd64/gocl main.go
