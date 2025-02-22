#pragma once

#include <string>
#include <stdexcept>
#include <iostream>

template<typename T>
void PrintValueCategory(T&& value)
{
	std::cout << "Value category: "
		<< (std::is_lvalue_reference<T>::value ? "lvalue" :
			std::is_rvalue_reference<T>::value ? "rvalue" : "unknown")
		<< " at address " << &value << std::endl;
}

class DestructorTest
{
public:
	static int destructorCount;
	static int copyCount;
	static int moveCount;

	DestructorTest()
	{
		std::cout << "Default constructor" << std::endl;
	}

	DestructorTest(const DestructorTest&)
	{
		copyCount++;
		std::cout << "Copy constructor called!" << std::endl;
	}

	DestructorTest(DestructorTest&&) noexcept
	{
		moveCount++;
		std::cout << "Move constructor called!" << std::endl;
	}

	DestructorTest& operator=(const DestructorTest&)
	{
		copyCount++;
		std::cout << "Copy assignment called!" << std::endl;
		return *this;
	}

	DestructorTest& operator=(DestructorTest&&) noexcept
	{
		moveCount++;
		std::cout << "Move assignment called!" << std::endl;
		return *this;
	}

	~DestructorTest()
	{
		destructorCount++;
		std::cout << "Destructor called!" << std::endl;
	}

	static void ResetCounters()
	{
		destructorCount = 0;
		copyCount = 0;
		moveCount = 0;
	}
};

