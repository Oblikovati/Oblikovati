#pragma once

namespace Oblikovati::Kernel
{
	/// @brief Provides an interface for a collection of elements that can be enumerated.
	/// Compatitle with C# IEnumerable interface.
	/// @tparam T The type of elements in the collection.
	template<typename T>
	class IEnumerable {
	public:
		virtual ~IEnumerable() = default;
		virtual size_t Count() const = 0;
		virtual const T& operator[](size_t index) const = 0;
		virtual T& operator[](size_t index) = 0;
	};

	/// @brief Provides an interface for an enumerator that can be used to iterate through a collection of elements.
	/// Compatitle with C# IEnumerator interface.
	/// @tparam TCollection The type of the collection to enumerate.
	/// @tparam T The type of elements in the collection.
	template<typename TCollection, typename T>
	class Enumerator {
	private:
		const TCollection* collection;
		size_t currentIndex;
	public:
		Enumerator(const TCollection* coll) : collection(coll), currentIndex(-1) {}

		bool MoveNext() {
			if (currentIndex + 1 < collection->Count()) {
				currentIndex++;
				return true;
			}
			return false;
		}

		void Reset() {
			currentIndex = -1;
		}

		T& Current() {
			if (currentIndex < 0 || currentIndex >= collection->Count()) {
				throw std::runtime_error("Invalid enumerator position");
			}
			return (*collection)[currentIndex];
		}

		const T& Current() const {
			if (currentIndex < 0 || currentIndex >= collection->Count()) {
				throw std::runtime_error("Invalid enumerator position");
			}
			return (*collection)[currentIndex];
		}
	};
}
