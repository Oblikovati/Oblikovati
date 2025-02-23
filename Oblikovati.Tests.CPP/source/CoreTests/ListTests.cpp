#include <gtest/gtest.h>
#include <string>
#include <stdexcept>
#include "OblikovatiKernel.h"
#include "DestructorTest.h"

namespace Oblikovati::Tests::Core {

	class ListTest : public ::testing::Test
	{
	protected:
		Oblikovati::Kernel::List<int> list;
		void SetUp() override
		{
			list.Clear();
		}
	};

	// Constructor Tests
	TEST_F(ListTest, DefaultConstructor_CreatesEmptyList)
	{
		EXPECT_EQ(list.Count(), 0);
	}

	// Add Tests
	TEST_F(ListTest, Add_SingleItem_IncreasesCount)
	{
		list.Add(1);
		EXPECT_EQ(list.Count(), 1);
		EXPECT_EQ(list[0], 1);
	}

	TEST_F(ListTest, Add_MultipleItems_IncreasesCount)
	{
		list.Add(1);
		list.Add(2);
		list.Add(3);
		EXPECT_EQ(list.Count(), 3);
		EXPECT_EQ(list[0], 1);
		EXPECT_EQ(list[1], 2);
		EXPECT_EQ(list[2], 3);
	}

	// Remove Tests
	TEST_F(ListTest, Remove_ExistingItem_RemovesAndReturnsTrue)
	{
		list.Add(1);
		list.Add(2);
		list.Add(3);
		EXPECT_TRUE(list.Remove(2));
		EXPECT_EQ(list.Count(), 2);
		EXPECT_EQ(list[0], 1);
		EXPECT_EQ(list[1], 3);
	}

	TEST_F(ListTest, Remove_NonExistingItem_ReturnsFalse)
	{
		list.Add(1);
		list.Add(2);
		EXPECT_FALSE(list.Remove(3));
		EXPECT_EQ(list.Count(), 2);
	}

	TEST_F(ListTest, Remove_FromEmptyList_ReturnsFalse)
	{
		EXPECT_FALSE(list.Remove(1));
	}

	// RemoveAt Tests
	TEST_F(ListTest, RemoveAt_ValidIndex_RemovesItem)
	{
		list.Add(1);
		list.Add(2);
		list.Add(3);
		list.RemoveAt(1);
		EXPECT_EQ(list.Count(), 2);
		EXPECT_EQ(list[0], 1);
		EXPECT_EQ(list[1], 3);
	}

	TEST_F(ListTest, RemoveAt_InvalidIndex_ThrowsException)
	{
		list.Add(1);
		EXPECT_THROW(list.RemoveAt(1), std::out_of_range);
		EXPECT_THROW(list.RemoveAt(-1), std::out_of_range);
	}

	// Clear Tests
	TEST_F(ListTest, Clear_NonEmptyList_RemovesAllItems)
	{
		list.Add(1);
		list.Add(2);
		list.Clear();
		EXPECT_EQ(list.Count(), 0);
	}

	TEST_F(ListTest, Clear_EmptyList_RemainsEmpty)
	{
		list.Clear();
		EXPECT_EQ(list.Count(), 0);
	}

	// Indexer Tests
	TEST_F(ListTest, Indexer_ValidIndex_ReturnsCorrectItem)
	{
		list.Add(1);
		list.Add(2);
		EXPECT_EQ(list[0], 1);
		EXPECT_EQ(list[1], 2);
	}

	TEST_F(ListTest, Indexer_InvalidIndex_ThrowsException)
	{
		list.Add(1);
		EXPECT_THROW(list[1], std::out_of_range);
		EXPECT_THROW(list[-1], std::out_of_range);
	}

	// Iterator Tests
	TEST_F(ListTest, Iterator_EmptyList_NoIteration)
	{
		int count = 0;
		for (const auto& item : list)
		{
			count++;
		}
		EXPECT_EQ(count, 0);
	}

	TEST_F(ListTest, Iterator_NonEmptyList_IteratesAllItems)
	{
		list.Add(1);
		list.Add(2);
		list.Add(3);

		std::vector<int> items;
		for (const auto& item : list)
		{
			items.push_back(item);
		}

		EXPECT_EQ(items.size(), 3);
		EXPECT_EQ(items[0], 1);
		EXPECT_EQ(items[1], 2);
		EXPECT_EQ(items[2], 3);
	}

	// Copy Constructor Tests
	TEST_F(ListTest, CopyConstructor_CopiesAllItems)
	{
		list.Add(1);
		list.Add(2);

		Oblikovati::Kernel::List<int> list2(list);
		EXPECT_EQ(list2.Count(), 2);
		EXPECT_EQ(list2[0], 1);
		EXPECT_EQ(list2[1], 2);
	}

	// Assignment Operator Tests
	TEST_F(ListTest, AssignmentOperator_CopiesAllItems)
	{
		list.Add(1);
		list.Add(2);

		Oblikovati::Kernel::List<int> list2;
		list2 = list;
		EXPECT_EQ(list2.Count(), 2);
		EXPECT_EQ(list2[0], 1);
		EXPECT_EQ(list2[1], 2);
	}

	// Capacity Tests
	TEST_F(ListTest, Capacity_GrowsAutomatically)
	{
		for (int i = 0; i < 100; i++)
		{
			list.Add(i);
		}
		EXPECT_EQ(list.Count(), 100);
		for (int i = 0; i < 100; i++)
		{
			EXPECT_EQ(list[i], i);
		}
	}


	TEST_F(ListTest, Destructor_ProperlyDestructsValues)
	{
		DestructorTest::ResetCounters();
		{
			Oblikovati::Kernel::List<DestructorTest> destructorDict;

			std::cout << "\nAdding first object:\n";
			DestructorTest test1;
			std::cout << "test1 address: " << &test1 << std::endl;

			auto&& movedTest1 = std::move(test1);
			std::cout << "Moved test1 address: " << &movedTest1 << std::endl;
			PrintValueCategory(std::move(movedTest1));

			std::cout << "Calling Add with move:\n";
			destructorDict.Add(std::move(movedTest1));

			std::cout << "\nAdding second object:\n";
			DestructorTest test2;
			std::cout << "test2 address: " << &test2 << std::endl;

			auto&& movedTest2 = std::move(test2);
			std::cout << "Moved test2 address: " << &movedTest2 << std::endl;
			PrintValueCategory(std::move(movedTest2));

			std::cout << "Calling Add with move:\n";
			destructorDict.Add(std::move(movedTest2));

			std::cout << "\nBefore list destruction:\n";
		}
		std::cout << "\nAfter list destruction:\n";

		std::cout << "Final counts - Destructor: " << DestructorTest::destructorCount
			<< ", Copy: " << DestructorTest::copyCount
			<< ", Move: " << DestructorTest::moveCount << std::endl;
	}

	// Move Semantics Tests
	TEST_F(ListTest, MoveConstructor_TransfersOwnership)
	{
		list.Add(1);
		list.Add(2);

		Oblikovati::Kernel::List<int> list2(std::move(list));
		EXPECT_EQ(list2.Count(), 2);
		EXPECT_EQ(list2[0], 1);
		EXPECT_EQ(list2[1], 2);
		EXPECT_EQ(list.Count(), 0);  // Original list should be empty
	}

	TEST_F(ListTest, MoveAssignment_TransfersOwnership)
	{
		list.Add(1);
		list.Add(2);

		Oblikovati::Kernel::List<int> list2;
		list2 = std::move(list);
		EXPECT_EQ(list2.Count(), 2);
		EXPECT_EQ(list2[0], 1);
		EXPECT_EQ(list2[1], 2);
		EXPECT_EQ(list.Count(), 0);  // Original list should be empty
	}

	// Edge Cases
	TEST_F(ListTest, Add_AfterClear_WorksCorrectly)
	{
		list.Add(1);
		list.Clear();
		list.Add(2);
		EXPECT_EQ(list.Count(), 1);
		EXPECT_EQ(list[0], 2);
	}

	TEST_F(ListTest, Remove_LastItem_WorksCorrectly)
	{
		list.Add(1);
		EXPECT_TRUE(list.Remove(1));
		EXPECT_EQ(list.Count(), 0);
	}

	// Performance Tests
	TEST_F(ListTest, Performance_QuickAdd)
	{
		const int itemCount = 100000;

		auto start = std::chrono::high_resolution_clock::now();

		for (int i = 0; i < itemCount; i++)
		{
			list.Add(i);
		}

		auto end = std::chrono::high_resolution_clock::now();
		auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);

		EXPECT_LT(duration.count(), 1);  // Should complete in less than 1 mili second
		EXPECT_EQ(list.Count(), itemCount);
	}

};
