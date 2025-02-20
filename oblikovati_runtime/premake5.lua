project "oblikovati_runtime"
    kind "ConsoleApp"

    debuggertype "NativeWithManagedCore"

	targetdir ("../bin/" .. outputdir .. "/%{prj.name}")
	objdir ("../bin-int/" .. outputdir .. "/%{prj.name}")

	links { "oblikovati_kernel" }

	defines { "GLM_FORCE_DEPTH_ZERO_TO_ONE", }

	files  { 
		"source/**.h",
		"source/**.c",
		"source/**.hpp",
		"source/**.cpp",
		
		"resources/shaders/**.glsl",
		"resources/shaders/**.glslh",
		"resources/shaders/**.hlsl",
		"resources/shaders/**.hlslh",
		"resources/shaders/**.slh",
	}

	includedirs  {
		"source/",
		"../oblikovati_kernel/source/"
	}

	filter "system:windows"
		systemversion "latest"

		defines { "OBKVT_PLATFORM_WINDOWS" }

		postbuildcommands {
			'{COPY} "../lib_vendor_precompiled/NvidiaAftermath/lib/x64/windows/GFSDK_Aftermath_Lib.x64.dll" "%{cfg.targetdir}"',
		}

	filter "system:linux"
		defines { "OBKVT_PLATFORM_LINUX", "__EMULATE_UUID", "BACKWARD_HAS_DW", "BACKWARD_HAS_LIBUNWIND" }
		links { "dw", "dl", "unwind", "pthread" }

		result, err = os.outputof("pkg-config --libs gtk+-3.0")
		linkoptions { result }

	filter "configurations:Debug or configurations:Debug-AS"
		symbols "On"
		defines { "OBKVT_DEBUG" }

		ProcessDependencies("Debug")

	filter { "system:windows", "configurations:Debug-AS" }
		sanitize { "Address" }
		flags { "NoRuntimeChecks", "NoIncrementalLink" }

	filter "configurations:Release"
		optimize "On"
        vectorextensions "AVX2"
        isaextensions { "BMI", "POPCNT", "LZCNT", "F16C" }
		defines { "OBKVT_RELEASE", }

		ProcessDependencies("Release")

	filter "configurations:Debug or configurations:Debug-AS or configurations:Release"
		defines {
			"OBKVT_TRACK_MEMORY",			
            "JPH_DEBUG_RENDERER",
            "JPH_FLOATING_POINT_EXCEPTIONS_ENABLED",
            "JPH_EXTERNAL_PROFILE"
		}

	filter "files:**.hlsl"
		flags {"ExcludeFromBuild"}

    filter "configurations:Dist"
        flags { "ExcludeFromBuild" }
