# Application Closing full local run

All four scenarios and 25 steps passed against the current feature and its
native document fixtures. Input hashes matched before and after the run.
The test code matches application commit `e12f967ff`.

Fixture preparation used disposable copies, scripted file picker events and
existing sketch and component placement controls. The close buttons and dialog
Cancel used X11 pointer input. The assembly case used X11 Alt+F4 and Escape.
The assembly prompt capture shows the component, document name, three answer
buttons, and the message frequency setting.

The runner used Linux Xvfb at 1600 × 1200, Openbox and Vulkan Lavapipe. It
selected the full `application-closing` feature without a scenario filter.

```sh
cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```

The build and runtime were unsealed. This is a full local pass. Strict
verification still requires the provenance described in issue #3978.
