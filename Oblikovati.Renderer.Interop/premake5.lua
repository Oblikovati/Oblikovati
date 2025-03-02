OblikovatiRootDirectory = os.getenv("OBLIKOVATI_DIR")

project "Oblikovati.Renderer.Interop"
	kind "SharedLib"
	language "C#"
	dotnetframework "net9.0"
	clr "Unsafe"
	targetdir "%{OblikovatiRootDirectory}/Oblikovati/Resources/Scripts"
	objdir "%{OblikovatiRootDirectory}/Oblikovati/Resources/Scripts/Intermediates"

	vsprops {
		AppendTargetFrameworkToOutputPath = "false",
		Nullable = "enable",
		CopyLocalLockFileAssemblies = "true",
		EnableDynamicLoading = "true",
	}

	links {
		"Oblikovati.API"
	}

	files {
		"%{OblikovatiRootDirectory}/Oblikovati.Renderer.Interop/Source/**.cs",
	}