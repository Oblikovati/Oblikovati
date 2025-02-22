#include "kernelpch.h"
#include "Log.h"

#include "spdlog/sinks/stdout_color_sinks.h"
#include "spdlog/sinks/basic_file_sink.h"

#include <filesystem>

#define OBKVT_HAS_CONSOLE !OBKVT_RELEASE

namespace Oblikovati {

	std::shared_ptr<spdlog::logger> Log::s_CoreLogger;
	std::shared_ptr<spdlog::logger> Log::s_ClientLogger;
	std::shared_ptr<spdlog::logger> Log::s_EditorConsoleLogger;

	std::map<std::string, Log::TagDetails> Log::s_DefaultTagDetails = {
		{ "AssetSystem",       TagDetails{  true, Level::Info  } },
		{ "Kernel",              TagDetails{  true, Level::Trace } },
		{ "GLFW",              TagDetails{  true, Level::Error } },
		{ "Memory",            TagDetails{  true, Level::Error } },
		{ "Project",           TagDetails{  true, Level::Warn  } },
		{ "Renderer",          TagDetails{  true, Level::Info  } },
		{ "Scripting",         TagDetails{  true, Level::Warn  } },
		{ "Timer",             TagDetails{ false, Level::Trace } },
	};

	void Log::Init()
	{
		// Create "logs" directory if doesn't exist
		std::string logsDirectory = "logs";
		if (!std::filesystem::exists(logsDirectory))
			std::filesystem::create_directories(logsDirectory);

		std::vector<spdlog::sink_ptr> kernelSinks =
		{
			std::make_shared<spdlog::sinks::basic_file_sink_mt>("logs/KERNEL.log", true),
#if OBKVT_HAS_CONSOLE
			std::make_shared<spdlog::sinks::stdout_color_sink_mt>()
#endif
		};

		std::vector<spdlog::sink_ptr> appSinks =
		{
			std::make_shared<spdlog::sinks::basic_file_sink_mt>("logs/APP.log", true),
#if OBKVT_HAS_CONSOLE
			std::make_shared<spdlog::sinks::stdout_color_sink_mt>()
#endif
		};

		std::vector<spdlog::sink_ptr> editorConsoleSinks =
		{
			std::make_shared<spdlog::sinks::basic_file_sink_mt>("logs/APP.log", true),
#if OBKVT_HAS_CONSOLE
			//std::make_shared<EditorConsoleSink>(1),
			std::make_shared<spdlog::sinks::stdout_color_sink_mt>()
#endif
		};

		kernelSinks[0]->set_pattern("[%T] [%l] %n: %v");
		appSinks[0]->set_pattern("[%T] [%l] %n: %v");

#if OBKVT_HAS_CONSOLE
		kernelSinks[1]->set_pattern("%^[%T] %n: %v%$");
		appSinks[1]->set_pattern("%^[%T] %n: %v%$");
		for (auto sink : editorConsoleSinks)
			sink->set_pattern("%^%v%$");
#endif

		s_CoreLogger = std::make_shared<spdlog::logger>("KERNEL", kernelSinks.begin(), kernelSinks.end());
		s_CoreLogger->set_level(spdlog::level::trace);

		s_ClientLogger = std::make_shared<spdlog::logger>("APP", appSinks.begin(), appSinks.end());
		s_ClientLogger->set_level(spdlog::level::trace);

		s_EditorConsoleLogger = std::make_shared<spdlog::logger>("Console", editorConsoleSinks.begin(), editorConsoleSinks.end());
		s_EditorConsoleLogger->set_level(spdlog::level::trace);

		SetDefaultTagSettings();
	}

	void Log::Shutdown()
	{
		s_EditorConsoleLogger.reset();
		s_ClientLogger.reset();
		s_CoreLogger.reset();
		spdlog::drop_all();
	}

	void Log::SetDefaultTagSettings()
	{
		s_EnabledTags = s_DefaultTagDetails;
	}

}
