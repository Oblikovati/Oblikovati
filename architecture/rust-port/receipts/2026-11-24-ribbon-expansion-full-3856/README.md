# Full published Ribbon Panel Expansion run

The full feature passed 58 of 67 scenarios and 652 of 661 executed steps.
Nine scenarios stopped at an undefined step. Seven of those cases require
Shell behavior, including preview while an expansion remains open.
The other two cover Highlight New over the Tools Options expansion and
toggling Explore from an expanded Modify title. The nine failures are listed
in the [Cucumber report](cucumber.json) and [native log](native.log.gz).

The [snapshot](snapshot.json) recorded the published feature's SHA-256
`3941cd9f652313f9209c5c9fe64f881414f6939a36591d725798b8e73ba39345`.
All 54 recorded file hashes matched the published inputs after the run.
The [source revision report](source-revision.json) confirms that the local
snapshot remained clean and unchanged.

The application base was `741e50fadf271ef05ad36312b29ddabc29dd2c52`
with local full Tools Options and scenario fixture changes. The mapped
test executable SHA-256 was
`321329f886f4356faffabb88f8e63dfe4ea97e96830097304cac7413804d15ae`.
The run used Linux X11, Xvfb, Openbox and Vulkan Lavapipe. It used the
binary built by `cargo test --locked -p oblikovati-head --lib --no-run`.

```sh
tools/with-xvfb.sh sh -c '
  openbox >/tmp/oblikovati-ribbon-relocated.RcZVDl/openbox-expansion-current.log 2>&1 &
  export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
  export LP_NUM_THREADS=4
  export OBK_BDD_ROOT=/tmp/oblikovati-ribbon-relocated.RcZVDl/snapshot/checkout
  export OBK_BDD_FEATURE=ribbon-panel-expansion
  export OBK_BDD_REPORTS=/tmp/oblikovati-ribbon-relocated.RcZVDl/expansion-current
  exec target/debug/deps/oblikovati_head-422152f5f0046b19 \
    application_scenarios_use_the_native_window --nocapture
'
```

The [JUnit report](junit.xml) and [scenario arguments](scenario-arguments.json)
record the full run. See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856)
and [Martensite #554](https://github.com/Oblikovati/Martensite/issues/554).
