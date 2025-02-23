#include <gtest/gtest.h>
#include <string>
#include <stdexcept>
#include "OblikovatiKernel.h"
#include "DestructorTest.h"

namespace Oblikovati::Tests::Core {

	class DictionaryTest : public ::testing::Test
	{
	protected:
		Oblikovati::Kernel::Dictionary<std::string, int> dict;
		void SetUp() override
		{
			dict.Clear();

		}
	};

	TEST_F(DictionaryTest, DefaultConstructor_CreatesEmptyDictionary)
	{
		EXPECT_EQ(dict.Count(), 0);
	}

	TEST_F(DictionaryTest, Add_NewKey_AddsSuccessfully)
	{
		dict.Add("test", 1);
		EXPECT_EQ(dict.Count(), 1);
		EXPECT_EQ(dict["test"], 1);
	}

	TEST_F(DictionaryTest, Add_DuplicateKey_ThrowsException)
	{
		dict.Add("test", 1);
		EXPECT_THROW(dict.Add("test", 2), std::runtime_error);
	}

	TEST_F(DictionaryTest, Add_MultipleItems_IncreasesCount)
	{
		dict.Add("one", 1);
		dict.Add("two", 2);
		dict.Add("three", 3);
		EXPECT_EQ(dict.Count(), 3);
	}

	TEST_F(DictionaryTest, Indexer_ExistingKey_ReturnsCorrectValue)
	{
		dict.Add("test", 42);
		EXPECT_EQ(dict["test"], 42);
	}

	TEST_F(DictionaryTest, Indexer_NonExistingKey_CreatesNewEntry)
	{
		dict["test"] = 42;
		EXPECT_EQ(dict.Count(), 1);
		EXPECT_EQ(dict["test"], 42);
	}

	TEST_F(DictionaryTest, Indexer_ModifyExistingValue_UpdatesValue)
	{
		dict["test"] = 42;
		dict["test"] = 24;
		EXPECT_EQ(dict["test"], 24);
	}

	TEST_F(DictionaryTest, ConstIndexer_NonExistingKey_ThrowsException)
	{
		const Oblikovati::Kernel::Dictionary<std::string, int>& constDict = dict;
		EXPECT_THROW(constDict["nonexistent"], std::out_of_range);
	}

	TEST_F(DictionaryTest, Remove_ExistingKey_RemovesAndReturnsTrue)
	{
		dict.Add("test", 1);
		EXPECT_TRUE(dict.Remove("test"));
		EXPECT_EQ(dict.Count(), 0);
	}

	TEST_F(DictionaryTest, Remove_NonExistingKey_ReturnsFalse)
	{
		EXPECT_FALSE(dict.Remove("nonexistent"));
	}

	TEST_F(DictionaryTest, Remove_FromEmptyDictionary_ReturnsFalse)
	{
		EXPECT_FALSE(dict.Remove("test"));
	}

	TEST_F(DictionaryTest, Clear_NonEmptyDictionary_RemovesAllItems)
	{
		dict.Add("one", 1);
		dict.Add("two", 2);
		dict.Clear();
		EXPECT_EQ(dict.Count(), 0);
	}

	TEST_F(DictionaryTest, Clear_EmptyDictionary_RemainsEmpty)
	{
		dict.Clear();
		EXPECT_EQ(dict.Count(), 0);
	}

	TEST_F(DictionaryTest, ContainsKey_ExistingKey_ReturnsTrue)
	{
		dict.Add("test", 1);
		EXPECT_TRUE(dict.ContainsKey("test"));
	}

	TEST_F(DictionaryTest, ContainsKey_NonExistingKey_ReturnsFalse)
	{
		EXPECT_FALSE(dict.ContainsKey("nonexistent"));
	}

	TEST_F(DictionaryTest, TryGetValue_ExistingKey_ReturnsTrueAndValue)
	{
		dict.Add("test", 42);
		int value = 1;
		EXPECT_TRUE(dict.TryGetValue("test", value));
		EXPECT_EQ(value, 42);
	}

	TEST_F(DictionaryTest, TryGetValue_NonExistingKey_ReturnsFalse)
	{
		int value =1;
		EXPECT_FALSE(dict.TryGetValue("nonexistent", value));
	}

	TEST_F(DictionaryTest, Iterator_EmptyDictionary_NoIteration)
	{
		int count = 0;
		for (const auto& pair : dict)
		{
			count++;
		}
		EXPECT_EQ(count, 0);
	}

	TEST_F(DictionaryTest, Iterator_NonEmptyDictionary_IteratesAllItems)
	{
		dict.Add("one", 1);
		dict.Add("two", 2);
		dict.Add("three", 3);

		std::set<std::string> keys;
		std::set<int> values;

		for (const auto& pair : dict)
		{
			keys.insert(pair.first);
			values.insert(pair.second);
		}

		EXPECT_EQ(keys.size(), 3);
		EXPECT_EQ(values.size(), 3);
		EXPECT_TRUE(keys.find("one") != keys.end());
		EXPECT_TRUE(keys.find("two") != keys.end());
		EXPECT_TRUE(keys.find("three") != keys.end());
	}

	TEST_F(DictionaryTest, CopyConstructor_CopiesAllItems)
	{
		dict.Add("one", 1);
		dict.Add("two", 2);

		Oblikovati::Kernel::Dictionary<std::string, int> dict2(dict);
		EXPECT_EQ(dict2.Count(), 2);
		EXPECT_EQ(dict2["one"], 1);
		EXPECT_EQ(dict2["two"], 2);
	}

	TEST_F(DictionaryTest, AssignmentOperator_CopiesAllItems)
	{
		dict.Add("one", 1);
		dict.Add("two", 2);

		Oblikovati::Kernel::Dictionary<std::string, int> dict2;
		dict2 = dict;
		EXPECT_EQ(dict2.Count(), 2);
		EXPECT_EQ(dict2["one"], 1);
		EXPECT_EQ(dict2["two"], 2);
	}

	TEST_F(DictionaryTest, StressTest_ManyItems)
	{
		const int itemCount = 1000;
		for (int i = 0; i < itemCount; i++)
		{
			dict.Add("key" + std::to_string(i), i);
		}

		EXPECT_EQ(dict.Count(), itemCount);

		for (int i = 0; i < itemCount; i++)
		{
			EXPECT_EQ(dict["key" + std::to_string(i)], i);
		}
	}

	TEST_F(DictionaryTest, EdgeCase_KeyWithSpecialCharacters)
	{
		std::string specialKey = "!@#$%^&*()";
		dict.Add(specialKey, 42);
		EXPECT_EQ(dict[specialKey], 42);
	}

	TEST_F(DictionaryTest, EdgeCase_EmptyStringKey)
	{
		dict.Add("", 42);
		EXPECT_EQ(dict[""], 42);
	}

	TEST_F(DictionaryTest, Performance_QuickLookup)
	{
		const int itemCount = 100000;

		for (int i = 0; i < itemCount; i++)
		{
			dict.Add("key" + std::to_string(i), i);
		}

		auto start = std::chrono::high_resolution_clock::now();

		for (int i = 0; i < itemCount; i++)
		{
			int value;
			dict.TryGetValue("key" + std::to_string(i), value);
		}

		auto end = std::chrono::high_resolution_clock::now();
		auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);

		// This is a simple performance check - adjust the threshold based on your needs
		EXPECT_LT(duration.count(), 122); // Should complete in less than 122 miliseconds
	}

	
	TEST_F(DictionaryTest, Destructor_ProperlyDestructsValues)
	{
		DestructorTest::ResetCounters();
		{
			Oblikovati::Kernel::Dictionary<std::string, DestructorTest> destructorDict;

			std::cout << "\nAdding first object:\n";
			DestructorTest test1;
			std::cout << "test1 address: " << &test1 << std::endl;

			auto&& movedTest1 = std::move(test1);
			std::cout << "Moved test1 address: " << &movedTest1 << std::endl;
			PrintValueCategory(std::move(movedTest1));

			std::cout << "Calling Add with move:\n";
			destructorDict.Add("test1", std::move(movedTest1));

			std::cout << "\nAdding second object:\n";
			DestructorTest test2;
			std::cout << "test2 address: " << &test2 << std::endl;

			auto&& movedTest2 = std::move(test2);
			std::cout << "Moved test2 address: " << &movedTest2 << std::endl;
			PrintValueCategory(std::move(movedTest2));

			std::cout << "Calling Add with move:\n";
			destructorDict.Add("test2", std::move(movedTest2));

			std::cout << "\nBefore dictionary destruction:\n";
		}
		std::cout << "\nAfter dictionary destruction:\n";

		std::cout << "Final counts - Destructor: " << DestructorTest::destructorCount
			<< ", Copy: " << DestructorTest::copyCount
			<< ", Move: " << DestructorTest::moveCount << std::endl;
	}
};
