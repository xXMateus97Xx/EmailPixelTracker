// Not Go code: this file only keeps the admin panel (and its node_modules,
// where some npm packages ship .go files) out of the API module, so that
// `go build ./...` and `go test ./...` at the project root ignore it.
module admin

go 1.26
