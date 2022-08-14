package line_edit

import (
	"fmt"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func New(valueRef *string, le *LineEdit) *LineEdit {
	var refLe *walk.LineEdit
	le.AssignTo = &refLe
	le.OnTextChanged = func() {
		fmt.Println(refLe.Text())
		*valueRef = refLe.Text()
	}
	return le
}
