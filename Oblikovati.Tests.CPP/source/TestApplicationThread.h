#pragma once

#include "kernel/Thread.h"

namespace Oblikovati::Host 
{
	class TestApplicationThread : public Kernel::Thread
	{
	private:
		/* data */
	public:
		TestApplicationThread(/* args */) : Thread("TestApplication")
		{
		
		}
		~TestApplicationThread()
		{
		
		}
	};
	

	}
	
};
