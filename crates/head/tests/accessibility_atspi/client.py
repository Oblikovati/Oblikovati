import gi,time,subprocess
from keyboard import press_enter
gi.require_version('Atspi','2.0')
from gi.repository import Atspi
Atspi.init()
def controls():
    desktop=Atspi.get_desktop(0)
    for i in range(desktop.get_child_count()):
        app=desktop.get_child_at_index(i)
        for j in range(app.get_child_count()):
            window=app.get_child_at_index(j)
            children=[window.get_child_at_index(k) for k in range(window.get_child_count())]
            names=[c.get_name() for c in children]
            if 'Minimize' in names and 'Close' in names:
                return window,children
    return None,None
def await_controls(name):
    for _ in range(600):
        window,buttons=controls()
        if buttons and any(b.get_name()==name for b in buttons): return window,buttons
        time.sleep(.1)
    raise AssertionError('missing button '+name)
window,buttons=await_controls('Maximize')
for button in buttons:
    bounds=button.get_component_iface().get_extents(Atspi.CoordType.SCREEN)
    print(button.get_name(),button.get_role_name(),bounds.x,bounds.y,bounds.width,bounds.height,flush=True)
    assert button.get_role_name()=='button' and bounds.width>0 and bounds.height>0
for name in ['Minimize', 'Maximize', 'Close']:
    button = next(b for b in buttons if b.get_name() == name)
    assert button.get_component_iface().grab_focus(), name
    for _ in range(600):
        button.clear_cache()
        if button.get_state_set().contains(Atspi.StateType.FOCUSED):
            break
        time.sleep(.1)
    else:
        raise AssertionError('focus was not applied to ' + name)
    print('Focused', name, flush=True)
other = subprocess.Popen(['xmessage', '-timeout', '30', 'Focus transfer test'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
try:
    for _ in range(100):
        window.clear_cache()
        if not window.get_state_set().contains(Atspi.StateType.ACTIVE): break
        time.sleep(.1)
    else: raise AssertionError('test window did not take native focus')
    target = next(b for b in buttons if b.get_name() == 'Maximize')
    assert target.get_component_iface().grab_focus()
    for _ in range(600):
        window.clear_cache()
        target.clear_cache()
        if window.get_state_set().contains(Atspi.StateType.ACTIVE) and target.get_state_set().contains(Atspi.StateType.FOCUSED): break
        time.sleep(.1)
    else: raise AssertionError('focus request did not activate the application and target')
    print('Focused background application', flush=True)
finally:
    other.terminate()
    other.wait(timeout=5)
press_enter()
window,buttons=await_controls('Restore')
print('Maximize action produced Restore',flush=True)
restore=next(b for b in buttons if b.get_name()=='Restore')
assert restore.get_action_iface().do_action(0)
window,buttons=await_controls('Maximize')
close=next(b for b in buttons if b.get_name()=='Close')
assert close.get_action_iface().do_action(0)
print('Close action sent',flush=True)
