VULKAN_SDK = os.getenv("VULKAN_SDK")

-- Keep in alphabetical order
Dependencies = {
	Coral = {
		LibName = "Coral.Native",
        IncludeDir = "%{wks.location}/lib_vendor_source/Coral/Coral.Native/Include"
    },
	Freetype = {
		LibName = "freetype"
	},
	GLFW = {
		LibName = "GLFW",
		IncludeDir = "%{wks.location}/lib_vendor_source/GLFW/include",
	},
	GLM = {
		IncludeDir = "%{wks.location}/lib_vendor_source/glm",
	},
	ImGui = {
		LibName = "ImGui",
		IncludeDir = "%{wks.location}/lib_vendor_source/imgui",
	},
	MSDFAtlasGen = {
		LibName = "msdf-atlas-gen",
		IncludeDir = "%{wks.location}/lib_vendor_source/msdf-atlas-gen/msdf-atlas-gen",
	},
	MSDFGen = {
		LibName = "msdfgen",
		IncludeDir = "%{wks.location}/lib_vendor_source/msdf-atlas-gen/msdfgen",
	},
	NvidiaAftermath = {
		LibName = "GFSDK_Aftermath_Lib.x64",
		IncludeDir = "%{wks.location}/lib_vendor_precompiled/NvidiaAftermath/include",
		Windows = { LibDir = "%{wks.location}/lib_vendor_precompiled/NvidiaAftermath/lib/x64/windows/" },
		Linux = { LibDir = "%{wks.location}/lib_vendor_precompiled/NvidiaAftermath/lib/x64/linux/" },
	},
	STB = {
		IncludeDir = "%{wks.location}/lib_vendor_source/stb/include",
	},
	Tracy = {
		LibName = "Tracy",
		IncludeDir = "%{wks.location}/lib_vendor_source/tracy/tracy/public",
	},
	Vulkan = {
		Windows = {
			LibName = "vulkan-1",
			IncludeDir = "%{VULKAN_SDK}/Include/",
			LibDir = "%{VULKAN_SDK}/Lib/",
		},
		Linux =  {
			LibName = "vulkan",
			IncludeDir = "%{VULKAN_SDK}/include/",
			LibDir = "%{VULKAN_SDK}/lib/",
		},
	}
}

include "./lib_internal_source/premake/dependency_extensions.lua"
