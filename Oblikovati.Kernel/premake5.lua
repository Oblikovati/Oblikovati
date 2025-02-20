project "Oblikovati.Kernel"
	kind "StaticLib"
	dependson "Coral.Managed"

    debuggertype "NativeWithManagedCore"

	targetdir ("../bin/" .. outputdir .. "/%{prj.name}")
	objdir ("../bin-int/" .. outputdir .. "/%{prj.name}")

	pchheader "KernelPCH.h"
	pchsource "source/KernelPCH.cpp"

	files {
		"source/**.h",
		"source/**.c",
		"source/**.hpp",
		"source/**.cpp",

		"platform/" .. os.target() .. "/**.hpp",
		"platform/" .. os.target() .. "/**.cpp",

		"../lib_vendor_source/VulkanMemoryAllocator/**.h",
		"../lib_vendor_source/VulkanMemoryAllocator/**.cpp",

		"../lib_vendor_source/imgui/misc/cpp/imgui_stdlib.cpp",
		"../lib_vendor_source/imgui/misc/cpp/imgui_stdlib.h"
	}

	includedirs { 
		"source/", 
		"../lib_vendor_source/",
	}

	IncludeDependencies()

	defines { "GLM_FORCE_DEPTH_ZERO_TO_ONE" }

	filter "files:../lib_vendor_source/imgui/misc/cpp/imgui_stdlib.cpp"
	flags { "NoPCH" }

	filter "system:windows"
		systemversion "latest"
		defines { "OBKVT_PLATFORM_WINDOWS", }

	filter "system:linux"
		defines { "OBKVT_PLATFORM_LINUX", "__EMULATE_UUID", "BACKWARD_HAS_DW", "BACKWARD_HAS_LIBUNWIND" }
		links { "dw", "dl", "unwind", "pthread" }

	filter "configurations:Debug or configurations:Debug-AS"
		symbols "On"
		defines { "OBKVT_DEBUG", "_DEBUG", "ACL_ON_ASSERT_ABORT", }

	filter { "system:windows", "configurations:Debug-AS" }	
		sanitize { "Address" }
		flags { "NoRuntimeChecks", "NoIncrementalLink" }

	filter "configurations:Release"
		optimize "On"
        symbols "Off"
		vectorextensions "AVX2"
		isaextensions { "BMI", "POPCNT", "LZCNT", "F16C" }
		defines { "OBKVT_RELEASE", "NDEBUG", }

	filter { "configurations:Debug or configurations:Debug-AS or configurations:Release" }
		defines {
			"OBKVT_TRACK_MEMORY",
			"JPH_DEBUG_RENDERER",
			"JPH_FLOATING_POINT_EXCEPTIONS_ENABLED",
			"JPH_EXTERNAL_PROFILE"
		}

	filter { "system:windows", "configurations:Debug-AS" }
        postbuildcommands {
			'{MKDIR} "%{wks.location}/Oblikovati/DotNet"',
		    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Coral.Managed/Build/Debug-AS-%{cfg.system}/Coral.Managed.dll" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.dll"',
		    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Coral.Managed/Build/Debug-AS-%{cfg.system}/Coral.Managed.pdb" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.pdb"',
		    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Coral.Managed/Build/Debug-AS-%{cfg.system}/Coral.Managed.deps.json" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.deps.json"',
		 --   '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Coral.Managed/Build/Debug-AS-%{cfg.system}/Coral.Managed.runtimeconfig.json" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.runtimeconfig.json"',
	    }

    filter { "system:windows", "configurations:Debug" }
        postbuildcommands {
			'{MKDIR} "%{wks.location}/Oblikovati/DotNet"',
		    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Build/Debug/Coral.Managed.dll" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.dll"',
		    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Build/Debug/Coral.Managed.pdb" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.pdb"',
		    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Build/Debug/Coral.Managed.deps.json" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.deps.json"',
		--    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Build/Debug/Coral.Managed.runtimeconfig.json" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.runtimeconfig.json"',
	    }

    filter { "system:windows", "configurations:Release" }
        postbuildcommands {
			'{MKDIR} "%{wks.location}/Oblikovati/DotNet"',
		    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Build/Release/Coral.Managed.dll" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.dll"',
		    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Build/Release/Coral.Managed.pdb" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.pdb"',
		    '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Build/Release/Coral.Managed.deps.json" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.deps.json"',
		  --  '{COPYFILE} "%{wks.location}/lib_vendor_source/Coral/Build/Release/Coral.Managed.runtimeconfig.json" "%{wks.location}/Oblikovati/DotNet/Coral.Managed.runtimeconfig.json"',
	    }
