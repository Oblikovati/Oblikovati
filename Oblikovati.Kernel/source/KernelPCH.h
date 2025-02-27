#pragma once

#ifdef OBKVT_PLATFORM_WINDOWS
#include <Windows.h>
#endif

#include <algorithm>
#include <array>
#include <cmath>
#include <cstddef>
#include <cstdlib>
#include <cstdarg>
#include <csignal>
#include <filesystem>
#include <fstream>
#include <functional>
#include <limits>
#include <map>
#include <memory>
#include <mutex>
#include <random>
#include <set>
#include <string>
#include <string_view>
#include <type_traits>
#include <unordered_map>
#include <utility>
#include <vector>
#include <thread>
#include <stdexcept>
#include <optional>
#include <cstring>


#include <kernel/ObjectType.h>
#include <kernel/Object.h>
#include <kernel/Base.h>
#include <kernel/OblikovatiAssert.h>
#include <kernel/Log.h>
#include <kernel/Version.h>
#include <kernel/Enumerator.h>
#include <kernel/Iterator.h>
#include <kernel/List.h>
#include <kernel/Dictionary.h>
