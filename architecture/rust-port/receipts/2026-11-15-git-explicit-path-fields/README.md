# Git vendor explicit path metadata

The checker examined Cargo 1.99.0 vendor output for application source
`1abf91487195eee71738eacd5642344bb1a1b79b`. Cargo.lock SHA-256 was
`3e4636962747b157c0283b41f5ddaade0b5f0aad35b8da55b485ca591de4d20d`.
It checked 18 Git packages and 22,972 tracked files. The result remains partial;
see [observation.json](observation.json).

Three local Git fixtures with explicit README values true, false and a named
file, plus a direct license-file, passed after `cargo vendor --locked --offline`.
The regression suite rejects forged values and an added license-file even if
Cargo's vendor checksum file is updated. No native UI scenario was run.

Default and inherited paths, other manifest fields, compiler and runtime inputs
remain unproven. The result remains `complete: false` and `reusable: false`.
