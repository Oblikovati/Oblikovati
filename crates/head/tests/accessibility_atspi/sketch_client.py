# SPDX-License-Identifier: GPL-2.0-only
import time
import gi

gi.require_version('Atspi', '2.0')
from gi.repository import Atspi

Atspi.init()

def elements():
    desktop = Atspi.get_desktop(0)
    for app_index in range(desktop.get_child_count()):
        app = desktop.get_child_at_index(app_index)
        for window_index in range(app.get_child_count()):
            window = app.get_child_at_index(window_index)
            children = [window.get_child_at_index(i) for i in range(window.get_child_count())]
            if any(child.get_name() == 'Close' for child in children):
                return children
    return []

def wait_for(name, parent=None):
    for _ in range(600):
        candidates = elements() if parent is None else [parent.get_child_at_index(i) for i in range(parent.get_child_count())]
        found = next((child for child in candidates if child.get_name() == name), None)
        if found is not None:
            return found
        time.sleep(.1)
    raise AssertionError('missing accessible element: ' + name)

def invoke(element):
    for _ in range(100):
        element.clear_cache()
        action = element.get_action_iface()
        if action is not None:
            assert action.do_action(0), element.get_name()
            return
        time.sleep(.1)
    raise AssertionError('missing action interface: ' + element.get_name())

import os
answer = os.environ.get('OBLIKOVATI_SKETCH_ANSWER', 'Cancel')
dialog = wait_for('Exit environment')
assert dialog.get_role_name() == 'dialog'
assert 'sketch environment' in dialog.get_description()
for name in ['OK', 'Cancel']:
    button = wait_for(name, dialog)
    bounds = button.get_component_iface().get_extents(Atspi.CoordType.SCREEN)
    assert bounds.width > 0 and bounds.height > 0, name
invoke(wait_for(answer, dialog))
if answer == 'Cancel':
    for _ in range(100):
        if not any(child.get_name() == 'Exit environment' for child in elements()): break
        time.sleep(.1)
    else: raise AssertionError('Cancel did not dismiss the sketch prompt')
    for _ in range(100):
        close = wait_for('Close')
        if close.get_state_set().contains(Atspi.StateType.ENABLED): break
        time.sleep(.1)
    invoke(close)
    dialog = wait_for('Save changes')
    invoke(wait_for('No', dialog))
else:
    dialog = wait_for('Save changes')
    invoke(wait_for('No', dialog))
    for _ in range(100):
        close = wait_for('Close')
        if close.get_state_set().contains(Atspi.StateType.ENABLED): break
        time.sleep(.1)
    invoke(close)
print('Sketch-exit prompt: ' + answer + ' verified', flush=True)
