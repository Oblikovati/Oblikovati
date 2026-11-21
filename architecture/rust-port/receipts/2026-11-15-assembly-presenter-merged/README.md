# Shared assembly presenter checks

The viewport library passed 165 tests on 2026-11-15 against application commit
`f38af1d6aa717aa257a348a2fd9dbbda9afc563f`. This includes the six shared
assembly-presentation traversal tests. The test used an isolated Cargo target
and Xvfb. The existing local `README.md` edit did not affect the build.

```sh
CARGO_TARGET_DIR=/tmp/obk-template-merged-target XMODIFIERS=@im=none \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-viewport --lib -- --test-threads=1
```

An earlier run without a display had four X11 test failures; it was not used
as passing evidence. The shared traversal is an intermediate step. Express
loading, validated cache persistence and Load Full remain open in
[issue #3949](https://github.com/Oblikovati/Oblikovati/issues/3949).
