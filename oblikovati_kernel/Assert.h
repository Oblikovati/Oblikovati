#pragma once

#ifdef OBKVT_PLATFORM_WINDOWS
	#define OBKVT_DEBUG_BREAK __debugbreak()
#elif defined(OBKVT_COMPILER_CLANG)
	#define OBKVT_DEBUG_BREAK __builtin_debugtrap()
#else
	#define OBKVT_DEBUG_BREAK
#endif // OBKVT_PLATFORM_WINDOWS

#ifdef OBKVT_DEBUG1
	#define OBKVT_ENABLE_ASSERTS
#endif // OBKVT_DEBUG

#ifdef OBKVT_ENABLE_ASSERTS
#ifdef OBKVT_COMPILER_CLANG
#define OBKVT_CORE_ASSERT_MESSAGE_INTERNAL(...)  ::Oblikovati::Log::PrintAssertMessage(::Hazel::Log::Type::Core, "Assertion Failed", ##__VA_ARGS__)
#define OBKVT_ASSERT_MESSAGE_INTERNAL(...)  ::Oblikovati::Log::PrintAssertMessage(::Hazel::Log::Type::Client, "Assertion Failed", ##__VA_ARGS__)
#else
#define OBKVT_CORE_ASSERT_MESSAGE_INTERNAL(...)  ::Oblikovati::Log::PrintAssertMessage(::Hazel::Log::Type::Core, "Assertion Failed" __VA_OPT__(,) __VA_ARGS__)
#define OBKVT_ASSERT_MESSAGE_INTERNAL(...)  ::Oblikovati::Log::PrintAssertMessage(::Hazel::Log::Type::Client, "Assertion Failed" __VA_OPT__(,) __VA_ARGS__)
#endif

#define OBKVT_CORE_ASSERT(condition, ...) do { if(!(condition)) { HZ_CORE_ASSERT_MESSAGE_INTERNAL(__VA_ARGS__); HZ_DEBUG_BREAK; } } while(0)
#define OBKVT_ASSERT(condition, ...) do { if(!(condition)) { HZ_ASSERT_MESSAGE_INTERNAL(__VA_ARGS__); HZ_DEBUG_BREAK; } } while(0)
#else
#define OBKVT_CORE_ASSERT(condition, ...) ((void) (condition))
#define OBKVT_ASSERT(condition, ...) ((void) (condition))
#endif
