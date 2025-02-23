#pragma once

namespace Oblikovati::Kernel::UserInterface
{
	CONTRACT View : public Object
	{
		public:
		ObjectTypeEnum GetType() override { return kViewObject; }

		virtual void Update() = 0;
	};

	class ViewObject final : View
	{
		public:
		void Update() override
		{
			
		}
	};
}
