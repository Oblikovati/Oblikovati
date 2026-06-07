-- set_parameter: add a parameter, then change its expression and read the new
-- evaluated value back. Demonstrates parameters.set + parameters.get.
oblikovati.documents.create{ type = "part", name = "set-parameter-demo" }

oblikovati.parameters.add{ name = "Length", expression = "2 in" }
oblikovati.parameters.set{ name = "Length", expression = "3.5 in" }

local p = oblikovati.parameters.get{ name = "Length" }
print("Length = " .. p.value)
