# Reopen an already open Part

The pinned Document Opening scenario passed all nine steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. The Open dialog activated the
existing Part, created no extra tab, and left the drawing available. The
[capture](capture.png) shows the active Part and the remaining drawing tab.

The source was the clean snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. Its Document Opening feature
had SHA-256 `ce0be44dd1e0ded53c12de6a96465ffafca2307db3029351147c6d9606bf4b8b`.
The [source report](source-revision.json) confirms the source remained
unchanged. The application build used `741e50fadf271ef05ad36312b29ddabc29dd2c52`
with local GUI fixture and ribbon test edits. The locked API and Martensite
revisions were `907cd4c28713b47501b077b8d902dbb6a008aada` and
`bef25f47548431c8f128f08f66a1f719bcee4982`.

The test used the application test binary from
`cargo test --locked -p oblikovati-head --lib`. The private display
ran Openbox and the following environment selected the case.

```sh
export XMODIFIERS=@im=none
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
export LP_NUM_THREADS=4
export OBK_BDD_FEATURE=document-opening
export OBK_BDD_SCENARIO='Activate an already open part through the Open dialog'
export OBK_BDD_REPORTS=/tmp/oblikovati-ribbon-relocated.RcZVDl/pinned-open
target/debug/deps/oblikovati_head-422152f5f0046b19 \
  application_scenarios_use_the_native_window --nocapture
```

The [Cucumber report](cucumber.json), [JUnit report](junit.xml),
[scenario arguments](scenario-arguments.json), and [native log](native.log)
cover one pinned case. The [separate X11 input test](native-x11.log) also
passed the existing-Part activation and tab-count checks.
Strict acceptance remains open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
See [#3889](https://github.com/Oblikovati/Oblikovati/issues/3889).
