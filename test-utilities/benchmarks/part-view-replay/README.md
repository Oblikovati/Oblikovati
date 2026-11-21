# Part view replay measurement

Issue: #4118. This is an unoptimized Rust test-build measurement, not a frame-rate test.
The application base was `35ad071c6`; the local API draft was `3977493`.
The benchmark is `measure_solid_view_metadata_replay` in
`crates/head/tests/owner/part_scene/replay_ownership_view.rs`.

The workload opens one named prism with the same certified resolution policy in
the desktop owner and replay service. It changes the Primary view's Publish
option 20 times and requests an exact part result after each edit. For the
baseline, the new host reuse route was temporarily removed. All other workload
inputs were the same.

```sh
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
cargo test --offline \
  --config 'patch."https://github.com/Oblikovati/Oblikovati.API.git".oblikovati-api.path="/tmp/obk-api-part-design-view-4118/rust"' \
  -p oblikovati-head --lib measure_solid_view_metadata_replay --no-run

/usr/bin/time -f 'peak_rss_kib=%M wall_s=%e' \
  /home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target/debug/deps/oblikovati_head-46b3bd2cf083a6e8 \
  measure_solid_view_metadata_replay --ignored --nocapture --test-threads=1
```

| Run | Baseline time (µs) | Reuse time (µs) | Baseline RSS (KiB) | Reuse RSS (KiB) |
| --- | ---: | ---: | ---: | ---: |
| 1 | 2,346,889 | 12,953 | 108,104 | 106,764 |
| 2 | 2,369,501 | 9,637 | 108,312 | 105,940 |
| 3 | 2,380,840 | 9,692 | 107,680 | 106,412 |

The baseline performed 20 complete part replays per run. The reuse route
performed none. All six runs passed. Peak RSS measures the entire test process;
it does not measure allocation count or establish a memory bound for larger
parts. Interactive 60 fps performance remains unverified.
