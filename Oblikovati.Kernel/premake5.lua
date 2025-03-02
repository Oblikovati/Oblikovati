OblikovatiRootDirectory = os.getenv("OBLIKOVATI_DIR")

project "Oblikovati.Kernel"
	kind "SharedLib"
	language "C#"
	dotnetframework "net9.0"
	clr "Unsafe"
	targetdir "%{OblikovatiRootDirectory}/Oblikovati/Resources/Scripts"
	objdir "%{OblikovatiRootDirectory}/Oblikovati/Resources/Scripts/Intermediates"

	links {
		"Oblikovati.API",
		"Oblikovati.Renderer.Interop"
	}

	vsprops {
		AppendTargetFrameworkToOutputPath = "false",
		Nullable = "enable",
		CopyLocalLockFileAssemblies = "true",
		EnableDynamicLoading = "true",
	}

	files {
		"%{OblikovatiRootDirectory}/Oblikovati.Kernel/Source/**.cs",
	}