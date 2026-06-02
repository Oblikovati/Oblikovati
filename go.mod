module github.com/Oblikovati/oblikovati

go 1.22

require (
	github.com/Oblikovati/api v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

// github.com/Oblikovati/api is the Apache-2.0 contract, now a sibling repo
// (../Oblikovati.API). It is resolved for local development via the go.work
// workspace at the repo root; no replace directive is committed here.
