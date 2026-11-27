# Sketch placement from the Part ribbon

Commit `e897d3886` lets an existing placement tool enter the only planar sketch
in the active Part. The X11 regression clicked Line from the Part ribbon and
pressed Escape before placing a point. The sketch ID, profile, and undo history
were checked. The capture shows the selected tool's prompt and the active
sketch. Focused owner tests also passed for single and ambiguous sketch targets.

The five-scenario run passed three scenarios and 31 of 33 executed steps.
Both outside-edit cases pass the enabled checks for Line, Circle, Arc and
Rectangle before reaching the missing Sketch Fillet command. The same
assertion also expects Sketch Text, which has no integrated authoring path.
Neither unsupported tool is exposed as an enabled command.

The saved source and fixture hashes match the live source before and after
the run. Full-suite input included scripted window events. The separate Line
regression used X11 mouse and keyboard input. Both ran under Xvfb and Lavapipe.
