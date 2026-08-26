// Isolated module for the echo test fixture: a minimal c-shared add-in built by
// the loader integration test. Under testdata/ so normal `go build ./...` ignores
// it; its own go.mod keeps the c-shared build self-contained (no head deps).
module echofixture

go 1.27.0
