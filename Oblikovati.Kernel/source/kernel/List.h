#pragma once

#include "Enumerator.h"
#include "Iterator.h"
#include <cstddef>
#include <stdexcept>

namespace Oblikovati::Kernel 
{
	template <typename T>
	class List : public IEnumerable<T>, public IIterable<T>
	{
	public:
		List() {}
		~List() {}
		//TODO: Check what we really want to expose to the API
		virtual Enumerator<List<T>, T> GetEnumerator() = 0;
		virtual void Add(const T& item) = 0;
		virtual void RemoveAt(size_t index) = 0;
		virtual bool Remove(const T& item) = 0;
		virtual void Clear() = 0;
	};


	/// @brief Provides a generic List of objects that can be individually 
	/// accessed by index. Compatible with a C# List class.
	/// @details Implements IEnumerable and IIterable interfaces.
	/// @tparam T The type of elements in the list.
	template <typename T>
	class ListObject : public List<T>
	{
	private:
		T* array;
		size_t capacity;
		size_t count;

		void resize(size_t newCapacity)
		{
			T* newArray = new T[newCapacity];
			for (size_t i = 0; i < count; i++)
			{
				newArray[i] = array[i];
			}
			delete[] array;
			array = newArray;
			capacity = newCapacity;
		}

	public:
		using value_type = T;
		using reference = T&;
		using const_reference = const T&;
		using iterator = typename IIterable<T>::iterator;
		using const_iterator = typename IIterable<T>::const_iterator;
		using size_type = size_t;
		using difference_type = std::ptrdiff_t;

		ListObject() : array(new T[4]), capacity(4), count(0) {}

		~ListObject()
		{
			delete[] array;
		}

		ListObject(const ListObject& other) : array(new T[other.capacity]), capacity(other.capacity), count(other.count)
		{
			for (size_t i = 0; i < count; i++)
			{
				array[i] = other.array[i];
			}
		}

		ListObject& operator=(const ListObject& other)
		{
			if (this != &other)
			{
				delete[] array;
				capacity = other.capacity;
				count = other.count;
				array = new T[capacity];
				for (size_t i = 0; i < count; i++)
				{
					array[i] = other.array[i];
				}
			}
			return *this;
		}

		Enumerator<List<T>, T> GetEnumerator() const
		{
			return Enumerator<List<T>, T>(this);
		}

		void Add(const T& item) override
		{
			if (count == capacity)
			{
				resize(capacity * 2);
			}
			array[count++] = item;
		}

		void RemoveAt(size_t index) override
		{
			if (index >= count)
			{
				throw std::out_of_range("Index out of range");
			}
			for (size_t i = index; i < count - 1; i++)
			{
				array[i] = array[i + 1];
			}
			count--;
		}

		bool Remove(const T& item) override
		{
			for (size_t i = 0; i < count; i++)
			{
				if (array[i] == item)
				{
					RemoveAt(i);
					return true;
				}
			}
			return false;
		}

		void Clear() override
		{
			count = 0;
		}

		size_t Count() const override
		{
			return count;
		}

		T& operator[](size_t index) override
		{
			if (index >= count)
			{
				throw std::out_of_range("Index out of range");
			}
			return array[index];
		}

		const T& operator[](size_t index) const override
		{
			if (index >= count)
			{
				throw std::out_of_range("Index out of range");
			}
			return array[index];
		}

		iterator begin() override { return iterator(array); }
		iterator end() override { return iterator(array + count); }
		const_iterator begin() const override { return const_iterator(array); }
		const_iterator end() const override { return const_iterator(array + count); }
		const_iterator cbegin() const override { return const_iterator(array); }
		const_iterator cend() const override { return const_iterator(array + count); }
	};

}
