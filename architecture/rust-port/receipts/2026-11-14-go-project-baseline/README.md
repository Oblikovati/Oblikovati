# Go project baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`persistence/project_file.go` reads and writes project name, workspace, library
roots and file locations. Its comments defer activation and richer project
settings. A search of `head/ui` found no matching project dialog or wizard.

On Linux, `go test ./persistence -run '^Test.*Project' -count=1 -v` passed four
project-file tests. They check YAML round-trip, readable format, extension and
schema rejection. They do not test project activation, a native dialog, wizard
navigation or cancellation. The historical Rust native receipts cover the
recorded dialog scenarios, but are stale after native test inputs changed. They
do not establish project-file creation or Go file compatibility.
