# Ribbon Command Availability full local run

Three of five scenarios and 31 of 33 executed steps passed. The source, support
files, and native assembly and drawing fixtures match `inputs.sha256` before
and after the run. The test and product code match commit `c79d5592a`.

The Line button shows its own tooltip during sketch editing. The assembly
scenario verifies the existing controls without activating one. The drawing
scenario switches to Place Views and Annotate, and the final capture shows
the sheet beside the expected annotation groups. These scenarios do not
invoke the component creation or annotation commands.

Both failed scenarios expect Create commands to be enabled outside sketch
edit. The first failed check is Line. The app requires an active sketch edit
before running it.

The runner used scripted window inputs on Linux Xvfb and Vulkan Lavapipe with
`OBK_BDD_FEATURE=ribbon-command-availability` and no scenario filter.

```sh
cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
