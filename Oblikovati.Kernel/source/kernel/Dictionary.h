#pragma once

namespace Oblikovati::Kernel 
{
	template<typename TKey, typename TValue>
		class Dictionary : public IEnumerable<std::pair<const TKey, TValue>>, public IIterable<std::pair<const TKey, TValue>>
	{
	private:
		struct Node
		{
			TKey key;
			TValue value;
			Node* next;

			Node(const TKey& k, const TValue& v) : key(k), value(v), next(nullptr) {}
		};

		static constexpr size_t INITIAL_CAPACITY = 16;
		static constexpr float LOAD_FACTOR_THRESHOLD = 0.75f;

		Node** buckets;
		size_t capacity;
		size_t count;
		mutable std::vector<std::pair<const TKey, TValue>> pairs_cache;
		mutable bool cache_valid;

		size_t GetHash(const TKey& key) const
		{
			return std::hash<TKey>{}(key) % capacity;
		}

		void UpdateCache() const
		{
			if (!cache_valid)
			{
				pairs_cache.clear();
				pairs_cache.reserve(count);
				for (size_t i = 0; i < capacity; i++)
				{
					Node* current = buckets[i];
					while (current != nullptr)
					{
						pairs_cache.emplace_back(current->key, current->value);
						current = current->next;
					}
				}
				cache_valid = true;
			}
		}

	public:
		using value_type = std::pair<const TKey, TValue>;
		using reference = value_type&;
		using const_reference = const value_type&;
		using iterator = typename IIterable<value_type>::iterator;
		using const_iterator = typename IIterable<value_type>::const_iterator;
		using size_type = size_t;

		Dictionary() : buckets(new Node* [INITIAL_CAPACITY]()), capacity(INITIAL_CAPACITY), count(0), cache_valid(false) {}

		Dictionary(size_t InitialCapacity) : buckets(new Node* [InitialCapacity]()), capacity(InitialCapacity), count(0), cache_valid(false) {}

		~Dictionary()
		{
			Clear();
			delete[] buckets;
		}

		// Copy constructor
		Dictionary(const Dictionary& other) : buckets(new Node* [other.capacity]()), capacity(other.capacity), count(0), cache_valid(false)
		{
			for (size_t i = 0; i < other.capacity; i++)
			{
				Node* current = other.buckets[i];
				while (current != nullptr)
				{
					Add(current->key, current->value);
					current = current->next;
				}
			}
		}

		// Move constructor
		Dictionary(Dictionary&& other) noexcept
			: buckets(other.buckets)
			, capacity(other.capacity)
			, count(other.count)
			, pairs_cache(std::move(other.pairs_cache))
			, cache_valid(other.cache_valid)
		{
			other.buckets = nullptr;
			other.capacity = 0;
			other.count = 0;
			other.cache_valid = false;
		}

		// Move assignment
		Dictionary& operator=(Dictionary&& other) noexcept
		{
			if (this != &other)
			{
				Clear();
				delete[] buckets;

				buckets = other.buckets;
				capacity = other.capacity;
				count = other.count;
				pairs_cache = std::move(other.pairs_cache);
				cache_valid = other.cache_valid;

				other.buckets = nullptr;
				other.capacity = 0;
				other.count = 0;
				other.cache_valid = false;
			}
			return *this;
		}

		// Assignment operator
		Dictionary& operator=(const Dictionary& other)
		{
			if (this != &other)
			{
				Clear();
				delete[] buckets;

				capacity = other.capacity;
				count = 0;
				cache_valid = false;
				buckets = new Node * [capacity]();

				for (size_t i = 0; i < other.capacity; i++)
				{
					Node* current = other.buckets[i];
					while (current != nullptr)
					{
						Add(current->key, current->value);
						current = current->next;
					}
				}
			}
			return *this;
		}

		void Add(const TKey& key, const TValue& value)
		{
			if (ContainsKey(key))
			{
				throw std::runtime_error("Key already exists in dictionary");
			}

			if (static_cast<float>(count + 1) / capacity > LOAD_FACTOR_THRESHOLD)
			{
				size_t newCapacity = capacity * 2;
				Node** newBuckets = new Node * [newCapacity]();

				for (size_t i = 0; i < capacity; i++)
				{
					Node* current = buckets[i];
					while (current != nullptr)
					{
						Node* next = current->next;
						size_t newHash = std::hash<TKey>{}(current->key) % newCapacity;
						current->next = newBuckets[newHash];
						newBuckets[newHash] = current;
						current = next;
					}
				}

				delete[] buckets;
				buckets = newBuckets;
				capacity = newCapacity;
			}

			size_t hash = GetHash(key);
			Node* newNode = new Node(key, value);
			newNode->next = buckets[hash];
			buckets[hash] = newNode;
			count++;
			cache_valid = false;
		}

		bool Remove(const TKey& key)
		{
			size_t hash = GetHash(key);
			Node* current = buckets[hash];
			Node* previous = nullptr;

			while (current != nullptr)
			{
				if (current->key == key)
				{
					if (previous == nullptr)
					{
						buckets[hash] = current->next;
					}
					else
					{
						previous->next = current->next;
					}
					delete current;
					count--;
					cache_valid = false;
					return true;
				}
				previous = current;
				current = current->next;
			}
			return false;
		}

		void Clear()
		{
			for (size_t i = 0; i < capacity; i++)
			{
				Node* current = buckets[i];
				while (current != nullptr)
				{
					Node* next = current->next;
					delete current;
					current = next;
				}
				buckets[i] = nullptr;
			}
			count = 0;
			cache_valid = false;
			pairs_cache.clear();
		}

		bool ContainsKey(const TKey& key) const
		{
			size_t hash = GetHash(key);
			Node* current = buckets[hash];
			while (current != nullptr)
			{
				if (current->key == key)
				{
					return true;
				}
				current = current->next;
			}
			return false;
		}

		bool TryGetValue(const TKey& key, TValue& value) const
		{
			size_t hash = GetHash(key);
			Node* current = buckets[hash];
			while (current != nullptr)
			{
				if (current->key == key)
				{
					value = current->value;
					return true;
				}
				current = current->next;
			}
			return false;
		}

		iterator begin() override
		{
			UpdateCache();
			return iterator(pairs_cache.empty() ? nullptr : &pairs_cache[0]);
		}

		iterator end() override
		{
			UpdateCache();
			return iterator(pairs_cache.empty() ? nullptr : &pairs_cache[0] + pairs_cache.size());
		}

		const_iterator begin() const override
		{
			UpdateCache();
			return const_iterator(pairs_cache.empty() ? nullptr : &pairs_cache[0]);
		}

		const_iterator end() const override
		{
			UpdateCache();
			return const_iterator(pairs_cache.empty() ? nullptr : &pairs_cache[0] + pairs_cache.size());
		}

		const_iterator cbegin() const override
		{
			return begin();
		}

		const_iterator cend() const override
		{
			return end();
		}

		size_t Count() const override
		{
			return count;
		}

		value_type& operator[](size_t index) override
		{
			UpdateCache();
			if (index >= count)
			{
				throw std::out_of_range("Index out of range");
			}
			return pairs_cache[index];
		}

		const value_type& operator[](size_t index) const override
		{
			UpdateCache();
			if (index >= count)
			{
				throw std::out_of_range("Index out of range");
			}
			return pairs_cache[index];
		}

		TValue& operator[](const TKey& key)
		{
			size_t hash = GetHash(key);
			Node* current = buckets[hash];
			while (current != nullptr)
			{
				if (current->key == key)
				{
					return current->value;
				}
				current = current->next;
			}

			Add(key, TValue());
			current = buckets[hash];
			while (current != nullptr)
			{
				if (current->key == key)
				{
					return current->value;
				}
				current = current->next;
			}
			throw std::runtime_error("Unexpected error in operator[]");
		}

		const TValue& operator[](const TKey& key) const
		{
			size_t hash = GetHash(key);
			Node* current = buckets[hash];
			while (current != nullptr)
			{
				if (current->key == key)
				{
					return current->value;
				}
				current = current->next;
			}
			throw std::out_of_range("Key not found in dictionary");
		}

		Enumerator<Dictionary<TKey, TValue>, std::pair<const TKey, TValue>> GetEnumerator() const
		{
			return Enumerator<Dictionary<TKey, TValue>, std::pair<const TKey, TValue>>(this);
		}
	};
}
