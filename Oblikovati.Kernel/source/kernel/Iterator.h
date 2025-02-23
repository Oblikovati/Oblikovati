#pragma once

namespace Oblikovati::Kernel
{


	template<typename T>
	class BaseIterator
	{
	protected:
		T* ptr;

	public:
		using iterator_category = std::forward_iterator_tag;
		using value_type = T;
		using difference_type = std::ptrdiff_t;
		using pointer = T*;
		using reference = T&;

		BaseIterator(T* p = nullptr) : ptr(p) {}

		reference operator*() const { return *ptr; }
		pointer operator->() const { return ptr; }

		bool operator==(const BaseIterator& other) const { return ptr == other.ptr; }
		bool operator!=(const BaseIterator& other) const { return ptr != other.ptr; }

		BaseIterator& operator++()
		{
			++ptr;
			return *this;
		}

		BaseIterator operator++(int)
		{
			BaseIterator tmp(*this);
			++ptr;
			return tmp;
		}
	};

	template<typename TCollection, typename T>
	class Iterator : public BaseIterator<T>
	{
	public:
		using BaseIterator<T>::BaseIterator;
	};

	template<typename TCollection, typename T>
	class ConstIterator : public BaseIterator<const T>
	{
	public:
		using BaseIterator<const T>::BaseIterator;
	};

	template<typename T>
	class IIterable
	{
	public:
		using iterator = Iterator<IIterable<T>, T>;
		using const_iterator = ConstIterator<IIterable<T>, T>;

		IIterable() = default;
		virtual ~IIterable() = default;
		IIterable(IIterable const&) = default;
		IIterable& operator =(IIterable const&) = default;
		IIterable(IIterable&&) = default;
		IIterable& operator=(IIterable&&) = default;

		virtual iterator begin() = 0;
		virtual iterator end() = 0;
		virtual const_iterator begin() const = 0;
		virtual const_iterator end() const = 0;
		virtual const_iterator cbegin() const = 0;
		virtual const_iterator cend() const = 0;
	};
}
