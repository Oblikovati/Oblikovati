-- mass_properties: build a 2 cm cube and read its physical properties (volume,
-- surface area, centre of mass). Demonstrates model.physicalProperties.
oblikovati.documents.create{ type = "part", name = "mass-properties-demo" }
oblikovati.sketch.create{ plane = "XY" }
oblikovati.sketch.rectangle{ sketchIndex = 0, width = "20 mm", height = "20 mm" }
oblikovati.features.add{ kind = "extrude", args = { sketchIndex = 0, profileIndex = 0, distance = "20 mm" } }

local p = oblikovati.model.physicalProperties()
print(string.format("volume = %.3f cm3", p.volume))
print(string.format("area = %.3f cm2", p.area))
print(string.format("centroid = (%.3f, %.3f, %.3f) cm", p.centroid[1], p.centroid[2], p.centroid[3]))
