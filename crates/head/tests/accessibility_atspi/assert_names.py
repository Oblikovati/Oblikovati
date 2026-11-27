# SPDX-License-Identifier: GPL-2.0-only
"""Check the current process's native AT-SPI frame tree."""
import sys
import time
import gi

gi.require_version('Atspi', '2.0')
from gi.repository import Atspi

coordinates = {'screen': Atspi.CoordType.SCREEN, 'window': Atspi.CoordType.WINDOW}[sys.argv[1]]
process_id = int(sys.argv[2])
expected = [tuple(map(int, sys.argv[3 + i * 4:7 + i * 4])) for i in range(3)]
Atspi.init()
for _ in range(100):
    desktop = Atspi.get_desktop(0)
    for index in range(desktop.get_child_count()):
        app = desktop.get_child_at_index(index)
        if app.get_process_id() != process_id:
            continue
        for index in range(app.get_child_count()):
            window = app.get_child_at_index(index)
            buttons = [window.get_child_at_index(i) for i in range(window.get_child_count())]
            names = [button.get_name() for button in buttons]
            if names[:3] != ['Minimize', 'Maximize', 'Close']:
                continue
            for button, bounds in zip(buttons[:3], expected):
                assert button.get_role_name() == 'button', button.get_name()
                rect = button.get_component_iface().get_extents(coordinates)
                actual = (rect.x, rect.y, rect.width, rect.height)
                assert all(abs(a - b) <= 2 for a, b in zip(actual, bounds)), (button.get_name(), actual, bounds)
            sys.exit(0)
    time.sleep(.1)
raise AssertionError('the application process did not expose the named frame buttons through AT-SPI')
