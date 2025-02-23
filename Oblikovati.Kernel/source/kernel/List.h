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
	private:
		T* array;
		size_t capacity;
		size_t count;

		void Resize(size_t newCapacity)
		{
			T* newArray = new T[newCapacity];
			for (size_t i = 0; i < count; i++)
			{
				newArray[i] = std::move(array[i]);
			}
			delete[] array;
			array = newArray;
			capacity = newCapacity;
		}

	public:
		// Standard container type definitions
		using value_type = T;
		using reference = value_type&;
		using const_reference = const value_type&;
		using iterator = typename IIterable<T>::iterator;
		using const_iterator = typename IIterable<T>::const_iterator;
		using size_type = size_t;

		List() :IIterable<T>(),  array(new T[4]), capacity(4), count(0) {}

		~List() override
		{
			delete[] array;
		}

		List(const List& other) : array(new T[other.capacity]), capacity(other.capacity), count(other.count)
		{
			for (size_t i = 0; i < count; i++)
			{
				array[i] = other.array[i];
			}
		}

		List& operator=(const List& other)
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

		List(List&& other) noexcept
			: array(other.array)
			, capacity(other.capacity)
			, count(other.count)
		{
			other.array = nullptr;
			other.capacity = 0;
			other.count = 0;
		}

		List& operator=(List&& other) noexcept
		{
			if (this != &other)
			{
				delete[] array;
				array = other.array;
				capacity = other.capacity;
				count = other.count;
				other.array = nullptr;
				other.capacity = 0;
				other.count = 0;
			}
			return *this;
		}

		void Add(const T& item)
		{
			if (count == capacity)
			{
				Resize(capacity * 2);
			}
			array[count++] = item;
		}

		void Add(T&& item)
		{
			if (count == capacity)
			{
				Resize(capacity * 2);
			}
			array[count++] = std::move(item);
		}

		bool Remove(const T& item)
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

		void RemoveAt(size_t index)
		{
			if (index >= count)
			{
				throw std::out_of_range("Index out of range");
			}
			for (size_t i = index; i < count - 1; i++)
			{
				array[i] = std::move(array[i + 1]);
			}
			count--;
		}

		void Clear()
		{
			count = 0;
		}

		// IEnumerable implementation
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

		// IIterable implementation
		iterator begin() override
		{
			return iterator(array);
		}

		iterator end() override
		{
			return iterator(array + count);
		}

		const_iterator begin() const override
		{
			return const_iterator(array);
		}

		const_iterator end() const override
		{
			return const_iterator(array + count);
		}

		const_iterator cbegin() const override
		{
			return const_iterator(array);
		}

		const_iterator cend() const override
		{
			return const_iterator(array + count);
		}

		// Enumerator support
		Enumerator<List<T>, T> GetEnumerator() const
		{
			return Enumerator<List<T>, T>(this);
		}
	};

}
