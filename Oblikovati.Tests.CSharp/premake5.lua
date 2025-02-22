OblikovatiRootDirectory = os.getenv("OBLIKOVATI_DIR")

include (path.join(OblikovatiRootDirectory, "Oblikovati.Kernel"))

project "Oblikovati.Tests.CSharp"
	kind "SharedLib"
	language "C#"
	dotnetframework "net9.0"
	clr "Unsafe"
	targetdir ("../bin-tests/" .. outputdir .. "/%{prj.name}")
	objdir ("../bin-tests-int/" .. outputdir .. "/%{prj.name}")

	links {
		"Oblikovati.Kernel",
        "Oblikovati.API"
	}

	files {
		"./source/**.cs",
	}