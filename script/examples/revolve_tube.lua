-- revolve_tube: revolve a rectangular profile offset from the Y axis into a tube
-- (annular cylinder). Demonstrates a closed line profile + the revolve feature.
-- The profile spans radius 1..3 cm and height 2 cm, so the volume is
-- pi*(3^2 - 1^2)*2 = 16*pi ~= 50.265 cm^3.
oblikovati.documents.create{ type = "part", name = "revolve-tube-demo" }
oblikovati.sketch.create{ plane = "XY" }

oblikovati.sketch.addEntity{ sketchIndex = 0, kind = "line", points = {{1, 0}, {3, 0}} }
oblikovati.sketch.addEntity{ sketchIndex = 0, kind = "line", points = {{3, 0}, {3, 2}} }
oblikovati.sketch.addEntity{ sketchIndex = 0, kind = "line", points = {{3, 2}, {1, 2}} }
oblikovati.sketch.addEntity{ sketchIndex = 0, kind = "line", points = {{1, 2}, {1, 0}} }

oblikovati.features.add{ kind = "revolve", args = { sketchIndex = 0, profileIndex = 0, angle = "360 deg" } }

local props = oblikovati.model.physicalProperties()
print(string.format("volume = %.3f cm3", props.volume))
