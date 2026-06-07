-- extrude_block: sketch a rectangle and extrude it into a solid block.
-- Demonstrates the sketch -> profile -> extrude feature flow. A 40x30 mm
-- rectangle extruded 10 mm is a 4x3x1 cm block = 12 cm^3.
oblikovati.documents.create{ type = "part", name = "extrude-block-demo" }
oblikovati.sketch.create{ plane = "XY" }
oblikovati.sketch.rectangle{ sketchIndex = 0, width = "40 mm", height = "30 mm" }

oblikovati.features.add{ kind = "extrude", args = { sketchIndex = 0, profileIndex = 0, distance = "10 mm" } }

local props = oblikovati.model.physicalProperties()
print(string.format("volume = %.3f cm3", props.volume))
