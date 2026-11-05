# Geometric sketch tools implementation

Date: 2026-11-05. Tracking: [#3816](https://github.com/Oblikovati/Oblikovati/issues/3816)
and [#3818](https://github.com/Oblikovati/Oblikovati/issues/3818).

## Implemented boundary

The Sketch ribbon exposes Horizontal, Vertical and Coincident tools. Native mouse
input resolves authored entities through the released Martensite 0.8 analytic
pick adapter. The tools use the existing typed constraint client and canonical
request DTO. The model owns solving, history and candidate validation.

Horizontal and Vertical accept one line or two points. Coincident accepts two
points, or a point and a line, circle or arc. A line midpoint pick retains the
existing Go midpoint policy. Its screen radius is separate from model precision.
Session selection supplies the operands and highlight state; tools do not keep
a second selection list. A repeated pick does not replace its first snap state.

One active tool owns each viewport event. Escape, tool replacement, sketch exit
and changed document history retire its transient picks. Camera gestures retain
their inputs. Successful creation clears picks and reports DOF. Failed creation
keeps the operands and shows the error without adopting an unhealthy candidate.

[ADR-0031](../decisions/ADR-0031-converged-geometric-authoring.md) requires the
existing convergence check before geometric constraint creation adopts a solved
candidate. DOF alone does not prove convergence. The original singular
perpendicular seed remains a refusal regression; its solver improvement is
tracked in [#3819](https://github.com/Oblikovati/Oblikovati/issues/3819).

## Evidence and limits

Owner tests cover line and point forms, midpoint state, circle and arc forms,
wrong entity kinds, duplicate picks, cancellation, replacement, stale history,
conflict refusal and undo/redo. Ribbon click tests exercise both display scales.
Model and canonical API regressions check unchanged state after numerical refusal
and preserve the original failing inputs.

A live desktop test used X11 mouse and keyboard input with the Go MCPBridge.
Horizontal changed DOF from 4 to 3 and Vertical changed it to 2. Coincident changed
a new point pair from DOF 4 to 2. Escape, selection highlights, undo/redo and save
passed. A fresh process reopened the healthy sketch with all three constraints
and empty transient selection. MCP screenshots were checked. Both desktops
closed normally.

This completes these three geometric tools. Dimensional constraint tools,
constraint marker selection and the full cube acceptance journey remain open.
A healthy DOF=2 test sketch is not a fully constrained cube profile. Solid adapter
migration still requires the tracked Martensite region and mass-property work.
