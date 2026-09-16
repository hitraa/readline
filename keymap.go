package readline

// Action is a function invoked when a key binding is triggered.
// It receives the Editor and LineBuffer, and returns whether the input session is done,
// the resulting line if completed, and any error.
type Action func(e *Editor, buf *LineBuffer) (done bool, line string, err error)

// KeyMap maps a Key to its corresponding Action handler.
type KeyMap map[Key]Action

// Clone creates a copy of the keymap.
func (km KeyMap) Clone() KeyMap {
	cp := make(KeyMap, len(km))
	for k, v := range km {
		cp[k] = v
	}
	return cp
}

// Bind associates a key with a custom action.
func (km KeyMap) Bind(k Key, action Action) {
	km[k] = action
}

// Unbind removes a key binding from the keymap.
func (km KeyMap) Unbind(k Key) {
	delete(km, k)
}
