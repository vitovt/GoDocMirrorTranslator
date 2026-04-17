//go:build !windows && !(linux && !android) && !(darwin && !ios)

package gui

func newDefaultPicker() picker {
	return dialogPicker{}
}
