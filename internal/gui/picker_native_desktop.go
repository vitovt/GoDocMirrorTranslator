//go:build (linux && !android) || (darwin && !ios) || windows

package gui

import (
	"errors"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	nativedialog "github.com/sqweek/dialog"
)

type nativeDesktopPicker struct {
	fallback   picker
	pickInput  func() (string, error)
	pickOutput func() (string, error)
}

func newDefaultPicker() picker {
	return nativeDesktopPicker{
		fallback:   dialogPicker{},
		pickInput:  pickNativeInputImage,
		pickOutput: pickNativeOutputDir,
	}
}

func (p nativeDesktopPicker) PickInputImage(parent fyne.Window, onPicked func(path string, err error)) {
	path, err := p.pickInput()
	p.handleSelection(parent, onPicked, path, err, p.fallback.PickInputImage)
}

func (p nativeDesktopPicker) PickOutputDir(parent fyne.Window, onPicked func(path string, err error)) {
	path, err := p.pickOutput()
	p.handleSelection(parent, onPicked, path, err, p.fallback.PickOutputDir)
}

func (p nativeDesktopPicker) handleSelection(
	parent fyne.Window,
	onPicked func(path string, err error),
	path string,
	err error,
	fallback func(parent fyne.Window, onPicked func(path string, err error)),
) {
	switch {
	case err == nil && strings.TrimSpace(path) == "":
		onPicked("", nil)
	case err == nil:
		onPicked(filepath.Clean(path), nil)
	case errors.Is(err, nativedialog.ErrCancelled):
		onPicked("", nil)
	default:
		fallback(parent, onPicked)
	}
}

func pickNativeInputImage() (string, error) {
	return nativedialog.File().
		Filter("Images", "jpg", "jpeg", "png", "webp").
		Title("Select Input Image").
		Load()
}

func pickNativeOutputDir() (string, error) {
	return nativedialog.Directory().
		Title("Select Output Folder").
		Browse()
}
