-include .env

APP     := multe_backend
BIN     := bin/$(APP)

dev: 
	go run main.go
lint:
	golangci-lint run

deploy:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v3 \
	go build -ldflags="-s" -trimpath -o $(BIN)
	scp $(BIN) $(SERVER):/tmp/$(APP)
	ssh -t $(SERVER) "sudo mv /tmp/$(APP) /opt/$(APP)/$(APP) \
		&& sudo chown root:root /opt/$(APP)/$(APP) \
		&& sudo systemctl restart $(APP) \
		&& journalctl -u $(APP) -n 5 --no-pager"