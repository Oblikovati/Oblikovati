-- SPDX-License-Identifier: GPL-2.0-only
-- Add a small closed circle to the existing 10 mm mid-plane of a 20 mm block.
-- Native point coordinates and radius are centimetres.
local sketch = oblikovati.sketch.create{workPlaneIndex=3}
oblikovati.sketch.addEntity{
  sketchIndex=sketch.sketchIndex,
  kind="circle",
  points={{-2,-1.25}},
  radius="2.5 mm"
}
print("mid-plane sketch", sketch.sketchIndex)
