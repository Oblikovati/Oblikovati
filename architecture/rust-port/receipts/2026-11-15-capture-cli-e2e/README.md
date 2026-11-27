# Capture CLI report ownership

On the clean task base `d7a4244ca`, the original capture CLI opened
`native-test.log` inside the report directory before starting the native test.
The report reservation rejected that nonempty directory. The red native run is
retained in `red.log.gz`; it published no receipt.

The runner now keeps its streamed log outside the report directory during the
child run. It publishes that log with exclusive creation after the child exits.
Unit tests cover an empty directory at child start, failure output, existing
files, and a concurrent writer. They do not overwrite earlier evidence.

The live check restored the pinned snapshot and ran the real capture CLI under
Xvfb and Lavapipe. The `Activate an already open part through the Open dialog`
scenario passed all nine steps. The capture CLI published the receipt, and the
independent receipt checker accepted it. The final image was inspected; it
shows the active part tab and the drawing tab. The run injects input through
the native frame loop, not the operating system.

```sh
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a -s '-screen 0 1600x1200x24' \
  python3 -B tools/bdd/live_capture_check.py \
  --output architecture/rust-port/receipts/2026-11-15-capture-cli-e2e \
  --renderer 'Vulkan Lavapipe under Xvfb'
```

The task branch had changes only in the BDD tools when the receipt was made;
the application and test inputs were unchanged. The full local gate remains
open for the API pin and broader BDD work (#3965, #3972).
