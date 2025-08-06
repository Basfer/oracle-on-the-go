go build -o builds/win10_amd64/gocl.exe main.go

SET GOOS=linux&&SET GOARCH=amd64&&go build -o builds/linux_amd64/gocl main.go

SET GOOS=solaris&&SET GOARCH=sparc64&&go build -o builds/solaris_sparc64/gocl main.go
