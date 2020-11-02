# scrapbee_backend

## compile

### Linux + amd64:
env GOARCH=amd64 go build -o scrapbee_backend_lnx -ldflags "-s -w" scrapbee_backend.go

### Win + amd64:
env GOOS=windows GOARCH=amd64 go build -o scrapbee_backend.exe -ldflags '-s -w' scrapbee_backend.go

### Mac + amd64:
env GOOS=darwin GOARCH=amd64 go build -o scrapbee_backend_mac -ldflags '-s -w' scrapbee_backend.go

### Freebsd + amd64:
env GOOS=freebsd GOARCH=amd64 go build -o scrapbee_backend_bsd -ldflags '-s -w' scrapbee_backend.go

### Linux + 386:
env GOARCH=386 go build -o scrapbee_backend -ldflags "-s -w" scrapbee_backend.go

### Win + 386:
env GOOS=windows GOARCH=386 go build -o scrapbee_backend.exe -ldflags '-s -w' scrapbee_backend.go

### Freebsd + 386:
env GOOS=freebsd GOARCH=386 go build -o scrapbee_backend -ldflags '-s -w' scrapbee_backend.go


