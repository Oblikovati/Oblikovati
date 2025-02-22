#pragma once

namespace Oblikovati::Kernel
{
	template<typename TCollection, typename T>
	class Iterator {
	private:
		T* ptr;
	public:
		using iterator_category = std::forward_iterator_tag;
		using value_type = T;
		using difference_type = std::ptrdiff_t;
		using pointer = T*;
		using reference = T&;

		Iterator(T* p) : ptr(p) {}

		reference operator*() { return *ptr; }
		pointer operator->() { return ptr; }
		
		Iterator& operator++() {
			++ptr;
			return *this;
		}
		
		Iterator operator++(int) {
			Iterator tmp = *this;
			++ptr;
			return tmp;
		}

		bool operator==(const Iterator& other) const { return ptr == other.ptr; }
		bool operator!=(const Iterator& other) const { return ptr != other.ptr; }
	};

	template<typename TCollection, typename T>
	class ConstIterator {
	private:
		const T* ptr;
	public:
		using iterator_category = std::forward_iterator_tag;
		using value_type = T;
		using difference_type = std::ptrdiff_t;
		using pointer = const T*;
		using reference = const T&;

		ConstIterator(const T* p) : ptr(p) {}

		reference operator*() const { return *ptr; }
		pointer operator->() const { return ptr; }
		
		ConstIterator& operator++() {
			++ptr;
			return *this;
		}
		
		ConstIterator operator++(int) {
			ConstIterator tmp = *this;
			++ptr;
			return tmp;
		}

		bool operator==(const ConstIterator& other) const { return ptr == other.ptr; }
		bool operator!=(const ConstIterator& other) const { return ptr != other.ptr; }
	};

	template<typename T>
	class IIterable {
	public:
		using iterator = Iterator<IIterable<T>, T>;
		using const_iterator = ConstIterator<IIterable<T>, T>;
		
		virtual ~IIterable() = default;
		virtual iterator begin() = 0;
		virtual iterator end() = 0;
		virtual const_iterator begin() const = 0;
		virtual const_iterator end() const = 0;
		virtual const_iterator cbegin() const = 0;
		virtual const_iterator cend() const = 0;
	};
}
