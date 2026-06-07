-- sketch_lines: draw a closed triangle from three connected lines, plus a
-- rectangle, on a new sketch. Demonstrates sketch.create, sketch.addEntity (line),
-- and sketch.rectangle. Coordinates are in centimetres (database units).
oblikovati.documents.create{ type = "part", name = "sketch-lines-demo" }
oblikovati.sketch.create{ plane = "XY" }

-- Triangle: each line starts where the previous ended, closing back to the origin.
oblikovati.sketch.addEntity{ sketchIndex = 0, kind = "line", points = {{0, 0}, {4, 0}} }
oblikovati.sketch.addEntity{ sketchIndex = 0, kind = "line", points = {{4, 0}, {2, 3}} }
oblikovati.sketch.addEntity{ sketchIndex = 0, kind = "line", points = {{2, 3}, {0, 0}} }

-- A 5x5 cm rectangle adds four more lines.
oblikovati.sketch.rectangle{ sketchIndex = 0, width = "5 cm", height = "5 cm" }

local ents = oblikovati.sketch.entities{ sketchIndex = 0 }
local lines = 0
for _, e in ipairs(ents.entities) do
  if e.kind == "line" then lines = lines + 1 end
end
print("lines = " .. lines)
