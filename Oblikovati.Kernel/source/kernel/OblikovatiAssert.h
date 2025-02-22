#pragma once

#include "kernel/Base.h"
#include "kernel/Log.h"

#ifdef OBKVT_PLATFORM_WINDOWS
	#define OBKVT_DEBUG_BREAK __debugbreak()
#elif defined(OBKVT_COMPILER_CLANG)
	#define OBKVT_DEBUG_BREAK __builtin_debugtrap()
#else
	#define OBKVT_DEBUG_BREAK
#endif // OBKVT_PLATFORM_WINDOWS

#ifdef OBKVT_DEBUG
	#define OBKVT_ENABLE_ASSERTS
#endif // OBKVT_DEBUG

#ifdef OBKVT_ENABLE_ASSERTS
#ifdef OBKVT_COMPILER_CLANG
#define OBKVT_CORE_ASSERT_MESSAGE_INTERNAL(...)  ::Oblikovati::Log::PrintAssertMessage(::Oblikovati::Log::Type::Core, "Assertion Failed", ##__VA_ARGS__)
#define OBKVT_ASSERT_MESSAGE_INTERNAL(...)  ::Oblikovati::Log::PrintAssertMessage(::Oblikovati::Log::Type::Client, "Assertion Failed", ##__VA_ARGS__)
#else
#define OBKVT_CORE_ASSERT_MESSAGE_INTERNAL(...)  ::Oblikovati::Log::PrintAssertMessage(::Oblikovati::Log::Type::Core, "Assertion Failed" __VA_OPT__(,) __VA_ARGS__)
#define OBKVT_ASSERT_MESSAGE_INTERNAL(...)  ::Oblikovati::Log::PrintAssertMessage(::Oblikovati::Log::Type::Client, "Assertion Failed" __VA_OPT__(,) __VA_ARGS__)
#endif

#define OBKVT_CORE_ASSERT(condition, ...) do { if(!(condition)) { OBKVT_CORE_ASSERT_MESSAGE_INTERNAL(__VA_ARGS__); OBKVT_DEBUG_BREAK; } } while(0)
#define OBKVT_ASSERT(condition, ...) do { if(!(condition)) { OBKVT_ASSERT_MESSAGE_INTERNAL(__VA_ARGS__); OBKVT_DEBUG_BREAK; } } while(0)
#else
#define OBKVT_CORE_ASSERT(condition, ...) ((void) (condition))
#define OBKVT_ASSERT(condition, ...) ((void) (condition))
#endif
