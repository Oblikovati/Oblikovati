// Isolated module for the UI-extension test fixture: a c-shared add-in built by the
// addinhost integration test. It stands in for an external add-in that adds a ribbon
// button over the host API and acts when the button is clicked. Under testdata/ so
// normal `go build ./...` ignores it; its own go.mod keeps the build self-contained.
module uifixture

go 1.27.0
