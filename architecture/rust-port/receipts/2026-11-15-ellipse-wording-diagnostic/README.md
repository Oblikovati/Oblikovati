# Ellipse diagnostic before binding update

The focused diagnostic ran seven `ellipse-0` scenarios against application
commit `af2e2b010` and pinned source
`e76cb749ed43c73a91c24854057da7c890041f51`. The ellipse feature SHA-256 was
`af53af88f7a32b441d76154fd75aeb36e68d4e673c47d21ecefa5be14cef5a6f`.

The five R4 cases (`ellipse-01`, `-03`, `-06`, `-07`, and `-08`) passed in
diagnostic mode. `ellipse-05` stopped at an unbound measurement step; its
later failures are tainted. `ellipse-09` had seven unbound steps and is tracked
in #3995. The report records all steps and their taint state.

The run used Xvfb, Lavapipe, and frame-loop input with `XMODIFIERS=@im=none`.
It is a diagnostic, not verified BDD acceptance; native build provenance
remains open in #3978. The report's source checkout was clean and unchanged.
