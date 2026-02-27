set shell := ["bash", "-c"]

# Вывести список доступных команд
default:
    @just --list

# Собрать бинарник проекта
build:
    go build -o bin/blog-engine ./cmd/blog-engine

# Сгенерировать статический сайт (html, sitemap, graph)
generate:
    go run ./cmd/blog-engine build

# Запустить dev-сервер с live reload (отслеживание изменений в content/)
serve:
    go run ./cmd/blog-engine serve

# Запустить тесты
test:
    go test -v ./...

# Очистить артефакты сборки
clean:
    rm -rf public/ bin/
