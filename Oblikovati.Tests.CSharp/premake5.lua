OblikovatiRootDirectory = os.getenv("OBLIKOVATI_DIR")

project "Oblikovati.Tests.CSharp"
	kind "SharedLib"
	language "C#"
	dotnetframework "net9.0"
	clr "Unsafe"
	targetdir ("../bin-tests/" .. outputdir .. "/%{prj.name}")
	objdir ("../bin-tests-int/" .. outputdir .. "/%{prj.name}")
	
	nuget {
		"Microsoft.NET.Test.Sdk:17.13.0",
		"Moq:4.20.72",
		"xunit:2.9.3",
		"xunit.runner.visualstudio:3.0.2"
	}

	links {
		"Oblikovati.Kernel"
	}

	files {
		"./source/**.cs",
	}