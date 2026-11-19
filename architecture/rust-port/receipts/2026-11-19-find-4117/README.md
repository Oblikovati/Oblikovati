# Model browser Find scenarios

Four focused native scenarios passed on 2026-11-19: sketch parameter criteria
(12 steps), pending value (12 steps), saved-search prompt cancellation (6 steps),
and picker cancellation (6 steps). The feature SHA-256 was
`603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.
They ran from the #4117 worktree before its commit with Xvfb, Vulkan lavapipe,
and `XMODIFIERS=@im=none`.

Two native X11 tests exercised text entry, button enablement, criterion
selection and both cancellation paths. The Customize import-picker Cancel
regression also passed. The captures show the empty input and both prompts;
the long selected property label is clipped at the default Find window width.
An unrelated malformed-import test stopped on an outdated version-message
expectation before reaching its Cancel step.

These focused reports are not a full-feature run or verified build provenance.
See issue #4117 and issue #3978.
