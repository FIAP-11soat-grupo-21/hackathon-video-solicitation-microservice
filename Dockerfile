#Etapa de build do projeto
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /src

COPY . .

RUN go mod download
RUN CGO_ENABLED=1 go build -buildvcs=false -o main.exe ./cmd/http/main.go

#Etapa de execução
FROM alpine:3.23 AS runtime

WORKDIR /app

COPY --from=builder /src/main.exe .
COPY --from=builder /src/.docker/config/load_env.sh .

RUN mkdir -p /app/data

ENTRYPOINT [ "./main.exe" ]