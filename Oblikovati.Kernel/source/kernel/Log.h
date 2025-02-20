#pragma once

#include "Base.h"

#include "spdlog/spdlog.h"

#include <map>
#include <memory>
#include <string>
#include <string_view>
#include <utility>

#define OBKVT_ASSERT_MESSAGE_BOX (!OBKVT_DIST && OBKVT_PLATFORM_WINDOWS)

#if OBKVT_ASSERT_MESSAGE_BOX
	#ifdef OBKVT_PLATFORM_WINDOWS
		#include <Windows.h>
	#endif
#endif

namespace Oblikovati {

	class Log
	{
	public:
		enum class Type : uint8_t
		{
			Core = 0, Client = 1
		};
		enum class Level : uint8_t
		{
			Trace = 0, Info, Warn, Error, Fatal
		};
		struct TagDetails
		{
			bool Enabled = true;
			Level LevelFilter = Level::Trace;
		};

	public:
		static void Init();
		static void Shutdown();

		inline static std::shared_ptr<spdlog::logger>& GetCoreLogger() { return s_CoreLogger; }
		inline static std::shared_ptr<spdlog::logger>& GetClientLogger() { return s_ClientLogger; }
		inline static std::shared_ptr<spdlog::logger>& GetEditorConsoleLogger() { return s_EditorConsoleLogger; }

		static bool HasTag(const std::string& tag) { return s_EnabledTags.find(tag) != s_EnabledTags.end(); }
		static std::map<std::string, TagDetails>& EnabledTags() { return s_EnabledTags; }
		static void SetDefaultTagSettings();

#if defined(OBKVT_PLATFORM_WINDOWS)
		template<typename... Args>
		static void PrintMessage(Log::Type type, Log::Level level, std::format_string<Args...> format, Args&&... args);
#else
		template<typename... Args>
		static void PrintMessage(Log::Type type, Log::Level level, const std::string_view format, Args&&... args);
#endif

		template<typename... Args>
		static void PrintMessageTag(Log::Type type, Log::Level level, std::string_view tag, std::format_string<Args...> format, Args&&... args);

		static void PrintMessageTag(Log::Type type, Log::Level level, std::string_view tag, std::string_view message);

		template<typename... Args>
		static void PrintAssertMessage(Log::Type type, std::string_view prefix, std::format_string<Args...> message, Args&&... args);

		static void PrintAssertMessage(Log::Type type, std::string_view prefix);

	public:
		// Enum utils
		static const char* LevelToString(Level level)
		{
			switch (level)
			{
				case Level::Trace: return "Trace";
				case Level::Info:  return "Info";
				case Level::Warn:  return "Warn";
				case Level::Error: return "Error";
				case Level::Fatal: return "Fatal";
			}
			return "";
		}
		static Level LevelFromString(std::string_view string)
		{
			if (string == "Trace") return Level::Trace;
			if (string == "Info")  return Level::Info;
			if (string == "Warn")  return Level::Warn;
			if (string == "Error") return Level::Error;
			if (string == "Fatal") return Level::Fatal;

			return Level::Trace;
		}

	private:
		static std::shared_ptr<spdlog::logger> s_CoreLogger;
		static std::shared_ptr<spdlog::logger> s_ClientLogger;
		static std::shared_ptr<spdlog::logger> s_EditorConsoleLogger;

		inline static std::map<std::string, TagDetails> s_EnabledTags;
		static std::map<std::string, TagDetails> s_DefaultTagDetails;
	};

}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Tagged logs (prefer these!)                                                                                      //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Core logging
#define OBKVT_CORE_TRACE_TAG(tag, ...) ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Trace, tag, __VA_ARGS__)
#define OBKVT_CORE_INFO_TAG(tag, ...)  ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Info, tag, __VA_ARGS__)
#define OBKVT_CORE_WARN_TAG(tag, ...)  ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Warn, tag, __VA_ARGS__)
#define OBKVT_CORE_ERROR_TAG(tag, ...) ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Error, tag, __VA_ARGS__)
#define OBKVT_CORE_FATAL_TAG(tag, ...) ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Fatal, tag, __VA_ARGS__)

// Client logging
#define OBKVT_TRACE_TAG(tag, ...) ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Trace, tag, __VA_ARGS__)
#define OBKVT_INFO_TAG(tag, ...)  ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Info, tag, __VA_ARGS__)
#define OBKVT_WARN_TAG(tag, ...)  ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Warn, tag, __VA_ARGS__)
#define OBKVT_ERROR_TAG(tag, ...) ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Error, tag, __VA_ARGS__)
#define OBKVT_FATAL_TAG(tag, ...) ::Oblikovati::Log::PrintMessageTag(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Fatal, tag, __VA_ARGS__)

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Core Logging
#define OBKVT_CORE_TRACE(...)  ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Trace, __VA_ARGS__)
#define OBKVT_CORE_INFO(...)   ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Info, __VA_ARGS__)
#define OBKVT_CORE_WARN(...)   ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Warn, __VA_ARGS__)
#define OBKVT_CORE_ERROR(...)  ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Error, __VA_ARGS__)
#define OBKVT_CORE_FATAL(...)  ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Core, ::Oblikovati::Log::Level::Fatal, __VA_ARGS__)

// Client Logging
#define OBKVT_TRACE(...)   ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Trace, __VA_ARGS__)
#define OBKVT_INFO(...)    ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Info, __VA_ARGS__)
#define OBKVT_WARN(...)    ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Warn, __VA_ARGS__)
#define OBKVT_ERROR(...)   ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Error, __VA_ARGS__)
#define OBKVT_FATAL(...)   ::Oblikovati::Log::PrintMessage(::Oblikovati::Log::Type::Client, ::Oblikovati::Log::Level::Fatal, __VA_ARGS__)

// Editor Console Logging Macros
#define OBKVT_CONSOLE_LOG_TRACE(...)   Oblikovati::Log::GetEditorConsoleLogger()->trace(__VA_ARGS__)
#define OBKVT_CONSOLE_LOG_INFO(...)    Oblikovati::Log::GetEditorConsoleLogger()->info(__VA_ARGS__)
#define OBKVT_CONSOLE_LOG_WARN(...)    Oblikovati::Log::GetEditorConsoleLogger()->warn(__VA_ARGS__)
#define OBKVT_CONSOLE_LOG_ERROR(...)   Oblikovati::Log::GetEditorConsoleLogger()->error(__VA_ARGS__)
#define OBKVT_CONSOLE_LOG_FATAL(...)   Oblikovati::Log::GetEditorConsoleLogger()->critical(__VA_ARGS__)

namespace Oblikovati {

#if defined(OBKVT_PLATFORM_WINDOWS)
	template<typename... Args>
	void Log::PrintMessage(Log::Type type, Log::Level level, std::format_string<Args...> format, Args&&... args)
#else
	template<typename... Args>
	void Log::PrintMessage(Log::Type type, Log::Level level, const std::string_view format, Args&&... args)
#endif
	{
		auto detail = s_EnabledTags[""];
		if (detail.Enabled && detail.LevelFilter <= level)
		{
			auto logger = (type == Type::Core) ? GetCoreLogger() : GetClientLogger();
			switch (level)
			{
			case Level::Trace:
				logger->trace(format, std::forward<Args>(args)...);
				break;
			case Level::Info:
				logger->info(format, std::forward<Args>(args)...);
				break;
			case Level::Warn:
				logger->warn(format, std::forward<Args>(args)...);
				break;
			case Level::Error:
				logger->error(format, std::forward<Args>(args)...);
				break;
			case Level::Fatal:
				logger->critical(format, std::forward<Args>(args)...);
				break;
			}
		}
	}


	template<typename... Args>
	void Log::PrintMessageTag(Log::Type type, Log::Level level, std::string_view tag, const std::format_string<Args...> format, Args&&... args)
	{
		auto detail = s_EnabledTags[std::string(tag)];
		if (detail.Enabled && detail.LevelFilter <= level)
		{
			auto logger = (type == Type::Core) ? GetCoreLogger() : GetClientLogger();
			std::string formatted = std::format(format, std::forward<Args>(args)...);
			switch (level)
			{
				case Level::Trace:
					logger->trace("[{0}] {1}", tag, formatted);
					break;
				case Level::Info:
					logger->info("[{0}] {1}", tag, formatted);
					break;
				case Level::Warn:
					logger->warn("[{0}] {1}", tag, formatted);
					break;
				case Level::Error:
					logger->error("[{0}] {1}", tag, formatted);
					break;
				case Level::Fatal:
					logger->critical("[{0}] {1}", tag, formatted);
					break;
			}
		}
	}


	inline void Log::PrintMessageTag(Log::Type type, Log::Level level, std::string_view tag, std::string_view message)
	{
		auto detail = s_EnabledTags[std::string(tag)];
		if (detail.Enabled && detail.LevelFilter <= level)
		{
			auto logger = (type == Type::Core) ? GetCoreLogger() : GetClientLogger();
			switch (level)
			{
				case Level::Trace:
					logger->trace("[{0}] {1}", tag, message);
					break;
				case Level::Info:
					logger->info("[{0}] {1}", tag, message);
					break;
				case Level::Warn:
					logger->warn("[{0}] {1}", tag, message);
					break;
				case Level::Error:
					logger->error("[{0}] {1}", tag, message);
					break;
				case Level::Fatal:
					logger->critical("[{0}] {1}", tag, message);
					break;
			}
		}
	}


	template<typename... Args>
	void Log::PrintAssertMessage(Log::Type type, std::string_view prefix, std::format_string<Args...> message, Args&&... args)
	{
		auto logger = (type == Type::Core) ? GetCoreLogger() : GetClientLogger();
		auto formatted = std::format(message, std::forward<Args>(args)...);
		logger->error("{0}: {1}", prefix, formatted);

#if OBKVT_ASSERT_MESSAGE_BOX
		MessageBoxA(nullptr, formatted.data(), "Oblikovati Assert", MB_OK | MB_ICONERROR);
#endif
	}


	inline void Log::PrintAssertMessage(Log::Type type, std::string_view prefix)
	{
		auto logger = (type == Type::Core) ? GetCoreLogger() : GetClientLogger();
		logger->error("{0}", prefix);
#if OBKVT_ASSERT_MESSAGE_BOX
		MessageBoxA(nullptr, "No message :(", "Oblikovati Assert", MB_OK | MB_ICONERROR);
#endif
	}
}
