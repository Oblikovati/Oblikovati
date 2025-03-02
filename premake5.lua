include "./lib_internal_source/premake/solution_items.lua"
include "dependencies.lua"


workspace "Oblikovati"
	configurations { "Debug", "Debug-AS", "Release" }
	startproject "Oblikovati.Tests.CSharp"
    conformancemode "On"

	language "C++"
	cppdialect "C++20"
	staticruntime "Off"

	solution_items { ".editorconfig" }

	flags { "MultiProcessorCompile" }

	-- Don't remove this. Please never use Annex K functions functions.
    -- Reasoning: https://www.open-std.org/jtc1/sc22/wg14/www/docs/n1967.htm
	defines {
		"_CRT_SECURE_NO_WARNINGS",
		"NOMINMAX",
		"SPDLOG_USE_STD_FORMAT",
		"_SILENCE_CXX17_CODECVT_HEADER_DEPRECATION_WARNING",
		"TRACY_ENABLE",
		"TRACY_ON_DEMAND",
		"TRACY_CALLSTACK=10",
	}


    filter "action:vs*"
        linkoptions { "/ignore:4099" } -- Disable no PDB found warning
        disablewarnings { "4068" } -- Disable "Unknown #pragma mark warning"

	filter "language:C++ or language:C"
		architecture "x86_64"

	filter "configurations:Debug or configurations:Debug-AS"
		optimize "Off"
		symbols "On"

	filter { "system:windows", "configurations:Debug-AS" }	
		--sanitize { "Address" }
		flags { "NoRuntimeChecks", "NoIncrementalLink" }

	filter "configurations:Release"
		optimize "Full"
		symbols "Off"

	filter "system:windows"
		buildoptions { "/EHsc", "/Zc:preprocessor", "/Zc:__cplusplus" }

outputdir = "%{cfg.buildcfg}-%{cfg.system}-%{cfg.architecture}"

group "0 - Dependencies"
--include "lib_vendor_source/Coral/Coral.Native"
--include "lib_vendor_source/Coral/Coral.Managed"
include "lib_vendor_source/GLFW"
include "lib_vendor_source/imgui"
include "lib_vendor_source/msdf-atlas-gen"
include "lib_vendor_source/googletest/googletest"
include "lib_vendor_source/tracy"
group ""

group "1 - Kernel"
include "./lib_internal_source/Oblikovati-API"
include "Oblikovati.Renderer"
include "Oblikovati.Renderer.Interop"
include "Oblikovati.Kernel"
group ""

group "2 - Runtime"
--include "Oblikovati"
group ""

group "3 - Tests"
include "Oblikovati.Tests.CSharp"
--include "Oblikovati.Tests.CPP"
group ""