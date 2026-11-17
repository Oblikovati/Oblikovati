# SPDX-License-Identifier: GPL-2.0-only
import time
import gi
from keyboard import press_enter

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
    assert element.get_action_iface().do_action(0), element.get_name()

close = wait_for('Close')
invoke(close)
dialog = wait_for('Save changes')
assert dialog.get_role_name() == 'dialog'
assert 'before closing' in dialog.get_description()
for name in ['Yes', 'No', 'Cancel']:
    button = wait_for(name, dialog)
    bounds = button.get_component_iface().get_extents(Atspi.CoordType.SCREEN)
    assert bounds.width > 0 and bounds.height > 0, name
cancel = wait_for('Cancel', dialog)
assert cancel.get_component_iface().grab_focus()
for _ in range(100):
    cancel.clear_cache()
    if cancel.get_state_set().contains(Atspi.StateType.FOCUSED): break
    time.sleep(.1)
else: raise AssertionError('Cancel did not receive keyboard focus')
press_enter()
for _ in range(100):
    if not any(child.get_name() == 'Save changes' for child in elements()):
        break
    time.sleep(.1)
else:
    raise AssertionError('Cancel did not dismiss the prompt')
for _ in range(100):
    close = wait_for('Close')
    if close.get_state_set().contains(Atspi.StateType.ENABLED):
        break
    time.sleep(.1)
else:
    raise AssertionError('Close did not become enabled after Cancel')
invoke(close)
dialog = wait_for('Save changes')
invoke(wait_for('No', dialog))
print('Guarded close: dialog, Cancel, and No verified', flush=True)
