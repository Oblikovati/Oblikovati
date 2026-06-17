module oblikovati.org

go 1.22

require (
	github.com/yuin/gopher-lua v1.1.1
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410
	gopkg.in/yaml.v3 v3.0.1
	oblikovati.org/api v0.22.0
)

require golang.org/x/text v0.22.0 // indirect

// oblikovati.org/api is the Apache-2.0 contract, now a sibling repo
// (../Oblikovati.API). It is resolved for local development via the go.work
// workspace at the repo root; no replace directive is committed here.
