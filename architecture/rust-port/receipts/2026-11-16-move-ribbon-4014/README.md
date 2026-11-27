# Move ribbon diagnostic (#4014)

With Xvfb and lavapipe, the native frame-loop tests opened Move from both the
collapsed and open Modify panels. No window resizing or model command bypass was
used. Run the panel tests with:

```sh
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --offline --locked -p oblikovati-head --lib \
  move_opens_from_the_ -- --test-threads=1 --nocapture
```

The focused Move scenarios used pinned snapshot
`e76cb749ed43c73a91c24854057da7c890041f51`. Each reached its Move setup,
then stopped at a different undefined step. The logs retain those failures.
They are diagnostic native frame-loop results, not verified BDD receipts or
OS-level input evidence. Build provenance remains unresolved (#3978).
