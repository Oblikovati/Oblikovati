# Published Point Text run with Arial and Tahoma

The full native run at `1932b4b71` passed 29 of 38 Point Text scenarios with
cleanup. It recorded 226 passing steps and nine undefined steps across nine
scenarios. The Rust test exited with status 101. The published feature, support
documents, and icon matched their starting hashes after the run.

Compared with the earlier Tahoma run, text-08 passed its retained-defaults
checks. Text-18 passed all five font and height steps. Text-30 passed all
eleven named-preset steps. The run used a private Fontconfig configuration
with Arial extracted from the Core Fonts archive and Tahoma from Wine.

- Text-01 and text-13 need corrected fixture or solver-count expectations.
- Text-19, text-20, text-23, text-25, and text-27 need font decoration
  metrics. Text-20 and text-27 also have fixture or count questions.
- Text-26 needs the text-box control, orientation, and background fill
  expectations clarified.
- Text-37 needs the entered height and the expected preview height to agree.

The later focused case-conversion test passed the first five steps of text-25.
That result has its own receipt and does not change the full-feature count.

```sh
OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios OBK_BDD_FEATURE=text \
  OBK_BDD_REPORTS=<reports-folder> \
  FONTCONFIG_FILE=<private-arial-and-tahoma-fontconfig> \
  VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
  XMODIFIERS=@im=none WINIT_UNIX_BACKEND=x11 \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

Font decoration is tracked in
[Martensite #578](https://github.com/Oblikovati/Martensite/issues/578).
Build and runtime provenance remain open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
