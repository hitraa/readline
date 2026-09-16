package readline

type undoState struct {
	text string
	pos  int
}

// UndoStack manages undo and redo history for LineBuffer.
type UndoStack struct {
	undo []undoState
	redo []undoState
	max  int
}

// NewUndoStack creates a new UndoStack with a max limit.
func NewUndoStack(max int) *UndoStack {
	if max <= 0 {
		max = 100
	}
	return &UndoStack{max: max}
}

// Save pushes the current buffer state to the undo stack and clears redo.
func (u *UndoStack) Save(buf *LineBuffer) {
	state := undoState{text: buf.String(), pos: buf.Pos()}
	if len(u.undo) > 0 {
		last := u.undo[len(u.undo)-1]
		if last.text == state.text {
			return // no change
		}
	}
	u.undo = append(u.undo, state)
	if len(u.undo) > u.max {
		u.undo = u.undo[len(u.undo)-u.max:]
	}
	u.redo = u.redo[:0]
}

// Undo restores the previous state, pushing the current state to redo.
func (u *UndoStack) Undo(buf *LineBuffer) bool {
	if len(u.undo) == 0 {
		return false
	}
	curr := undoState{text: buf.String(), pos: buf.Pos()}
	u.redo = append(u.redo, curr)

	prev := u.undo[len(u.undo)-1]
	u.undo = u.undo[:len(u.undo)-1]

	buf.Set(prev.text)
	buf.SetPos(prev.pos)
	return true
}

// Redo reapplies a previously undone state.
func (u *UndoStack) Redo(buf *LineBuffer) bool {
	if len(u.redo) == 0 {
		return false
	}
	curr := undoState{text: buf.String(), pos: buf.Pos()}
	u.undo = append(u.undo, curr)

	next := u.redo[len(u.redo)-1]
	u.redo = u.redo[:len(u.redo)-1]

	buf.Set(next.text)
	buf.SetPos(next.pos)
	return true
}

// Reset clears both undo and redo stacks.
func (u *UndoStack) Reset() {
	u.undo = u.undo[:0]
	u.redo = u.redo[:0]
}
