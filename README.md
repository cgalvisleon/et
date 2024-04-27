# Go begin init

```
go mod init github.com/cgalvisleon/et
```

## Dependencis

```
go get github.com/fsnotify/fsnotify
go get github.com/gorilla/websocket
go get github.com/mattn/go-colorable
go get github.com/dimiro1/banner
go get github.com/go-chi/chi/v5
go get github.com/shirou/gopsutil/v3/mem
```

## CDM

```
go run ./cmd/service/main.go
go run ./cmd/gateway/main.go -port 3300 -rpc 4200
```

## Condicion de carrera

```
go build --race ./cmd/gateway/main.go
go build --race ./cmd/serive/main.go

go build ./cmd/gateway/main.go
```
