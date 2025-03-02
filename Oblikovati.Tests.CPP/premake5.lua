project "Oblikovati.Tests.CPP"
    kind "ConsoleApp"
	
    debuggertype "NativeWithManagedCore"

	targetdir ("../bin/" .. outputdir .. "/%{prj.name}")
	objdir ("../bin-int/" .. outputdir .. "/%{prj.name}")

	links { "Oblikovati.Kernel"}

	nuget {
		"Microsoft.googletest.v140.windesktop.msvcstl.static.rt-dyn:1.8.1.7",
	}

	defines { 
		"GLM_FORCE_DEPTH_ZERO_TO_ONE", 
		"GLM_ENABLE_EXPERIMENTAL",
		"VR_BUILD_DLL",
        "GLFW_INCLUDE_VULKAN" }

	IncludeDependencies()

	files  { 
		"source/**.h",
		"source/**.c",
		"source/**.hpp",
		"source/**.cpp",
	}

	includedirs  {
		"source/",
		"../Oblikovati.Kernel/source/",
		"../lib_vendor_source/",
	}

	filter "system:windows"
		systemversion "latest"

		defines { "OBKVT_PLATFORM_WINDOWS" }

	filter "system:linux"
		defines { "OBKVT_PLATFORM_LINUX", "__EMULATE_UUID", "BACKWARD_HAS_DW", "BACKWARD_HAS_LIBUNWIND" }
		links { "dw", "dl", "unwind", "pthread" }

		result, err = os.outputof("pkg-config --libs gtk+-3.0")
		linkoptions { result }

	filter "configurations:Debug or configurations:Debug-AS"
		symbols "On"
		defines { "OBKVT_DEBUG" }

	filter { "system:windows", "configurations:Debug-AS" }
		--sanitize { "Address" }
		flags { "NoRuntimeChecks", "NoIncrementalLink" }
