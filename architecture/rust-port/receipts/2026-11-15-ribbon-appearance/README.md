# Ribbon appearance evidence

- Application under test: `5d1da5e2009fce581749a2bfcaf18e6f5514579e`.
- Scenario source: MCAD-Scenarios commit `48987af7b5cbb06515bc6f30385f508382f34064`.
- Feature SHA-256: `846aff6433de2b69e222aa0580140c53adb7fdbe3aeba04ccab2a69ddc8c9438`.
- The pinned checkout came from `test-utilities/benchmarks/scenario-snapshots/mcad-scenarios-2026-11-14-48987af.bundle`. The pinned feature matched the live feature before the run. The source revision report confirms that the pinned source did not change during the run.
- Platform: Linux x86-64, Xvfb, and the Mesa software Vulkan driver. The native harness sends pointer and keyboard events through the GUI frame input path.
- Martensite revision: `d051ff699982faba4ec2df3ee9a47e14f38dedbd`. API revision: `21b18a952fa7548a31e7eac338f6c78a32e5cfe1`.

The full `ribbon-appearance` feature ran with 16 of 18 scenarios and 186 of 188 steps passing. The two failures both require the missing Team Web fixture. The scenario owner has the fixture questions in that feature's `BLOCKERS.md`. The benchmark command returned a failure because of those two scenarios. The Cucumber and JUnit reports in this directory contain the full results.

The GUI library passed 490 tests with `--test-threads=1`. The 14 native appearance window tests passed, including real command selection, sketch state, settings reload, document types, and expanded panels. The attached images are from that window run. Strict Clippy passed for all GUI and head targets.

The later test-only change `701af9611` replaces a fixed-size iterator with its equivalent fixed-size slice. The icon size and row-count window test passed again after that change and after merging the later `rust-work` evidence commits.

Commands used the pinned source and dedicated no-debug Cargo target:

```sh
cargo test --locked -p oblikovati-gui --lib -- --test-threads=1
xvfb-run -a cargo test --locked -p oblikovati-head --lib modes_native -- --test-threads=1 --nocapture
xvfb-run -a cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --test-threads=1 --nocapture
cargo clippy --locked -p oblikovati-gui -p oblikovati-head --all-targets -- -D warnings
```

The window commands used `VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json` and `XMODIFIERS=@im=none`. The scenario command used `OBK_BDD_ROOT` for the pinned checkout, `OBK_BDD_FEATURE=ribbon-appearance`, and a dedicated `OBK_BDD_REPORTS` directory.
