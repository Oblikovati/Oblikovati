# Native scenario argument capture

On 2026-11-19, the focused model-browser scenario `Inspect existing feature
search criteria` passed all 13 steps under Xvfb and Vulkan lavapipe. The
runner's captured four-row table matched the current parsed source and the
SHA-256 of the Cucumber report. A separate comparison rejected a changed cell.
The feature SHA-256 was
`603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.

This was a focused test of the argument capture, not a full feature run or
strict verified acceptance. The scenario source was unversioned. See issue #4114
and issue #3978.
