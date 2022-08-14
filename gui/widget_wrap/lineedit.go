package widget_wrap

import (
	"github.com/lxn/walk"
)

type LineEditWrap struct {
	Value string
	OnTextChanged  func(string)
}

func New(le *walk.LineEdit) *LineEditWrap {
	lew := &LineEditWrap{
		Value: ""
	}

	le.OnTextChanged = func() {
		
	}
	return lew
}
