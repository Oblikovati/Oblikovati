# Auxiliary extent control for Extrude Cut

The source interaction captures a flipped cut and parenthesized
distance without establishing the original termination rule. The
desktop maps that control to its certified Through All Cut. The
parenthesized 10 mm value is the inactive numeric input. The named
solid determines the cut's reach.

The published case checks the control beside Distance A, the flipped
direction and cut preview, acceptance, and Undo. The dedicated
Through All and fixed-distance Cut cases also passed.

| Backend | Auxiliary Cut | Through All | Numeric Cut |
| --- | --- | --- | --- |
| Wayland | [10 passed](wayland-auxiliary.xml) | [8 passed](wayland-through-all.xml) | [9 passed](wayland-cut.xml) |
| X11 | [10 passed](x11-auxiliary.xml) | [8 passed](x11-through-all.xml) | [9 passed](x11-cut.xml) |

The full X11 run at the preset-default checkpoint, before the
sketch-resume and auxiliary changes, passed 47 of 74 Extrude cases.
Its 27 failures comprised 25 unbound cases and two failed assertions.
[Earlier full-run case results](earlier-x11-audit.tsv),
[published input checksums](published-inputs.sha256).
