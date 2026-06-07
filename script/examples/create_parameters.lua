-- create_parameters: add user parameters to a fresh part, one driven by an
-- expression in inches (stored in the document's units) and one in millimetres,
-- then list them back. Demonstrates parameters.add with unit-bearing expressions.
oblikovati.documents.create{ type = "part", name = "create-parameters-demo" }

oblikovati.parameters.add{ name = "Width", expression = "3 in" }
oblikovati.parameters.add{ name = "Height", expression = "40 mm" }

local ps = oblikovati.parameters.list()
for _, p in ipairs(ps.parameters) do
  print(p.name .. " = " .. p.value)
end
