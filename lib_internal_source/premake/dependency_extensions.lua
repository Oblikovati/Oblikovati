-- SPDX-FileCopyrightText: Hazel Engine https://github.com/StudioCherno/Hazel
--
-- SPDX-License-Identifier: Apache-2.0

include "ordered_pairs.lua"

function firstToUpper(str)
	return (str:gsub("^%l", string.upper))
end

function LinkDependency(table, is_debug, target)
	if table.LibDir ~= nil then
		libdirs { table.LibDir }
	end

	local libraryName = nil
	if table.LibName ~= nil then
		libraryName = table.LibName
	end

	if table.DebugLibName ~= nil and is_debug and target == "Windows" then
		libraryName = table.DebugLibName
	end

	if libraryName ~= nil then
		links { libraryName }
		return true
	end

	return false
end

function AddDependencyIncludes(table)
	if table.IncludeDir ~= nil then
		externalincludedirs { table.IncludeDir }
	end
end

function ProcessDependencies(config_name)
	local target = firstToUpper(os.target())

	for key, libraryData in orderedPairs(Dependencies) do
		local matchesConfiguration = true

		if config_name ~= nil and libraryData.Configurations ~= nil then
			matchesConfiguration = string.find(libraryData.Configurations, config_name)
		end

		local isDebug = config_name == "Debug"

		if matchesConfiguration then
			local continueLink = true

			if libraryData[target] ~= nil then
				continueLink = not LinkDependency(libraryData[target], isDebug, target)
				AddDependencyIncludes(libraryData[target])
			end

			if continueLink then
				LinkDependency(libraryData, isDebug, target)
			end

			AddDependencyIncludes(libraryData)
		end

	end
end

function IncludeDependencies(config_name)
	local target = firstToUpper(os.target())

	for key, libraryData in orderedPairs(Dependencies) do
		local matchesConfiguration = true

		if config_name ~= nil and libraryData.Configurations ~= nil then
			matchesConfiguration = string.find(libraryData.Configurations, config_name)
		end

		if matchesConfiguration then
			AddDependencyIncludes(libraryData)

			if libraryData[target] ~= nil then
				AddDependencyIncludes(libraryData[target])
			end
		end

	end
end