package gui

import (
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

type picker interface {
	PickInputImage(parent fyne.Window, onPicked func(path string, err error))
	PickOutputDir(parent fyne.Window, onPicked func(path string, err error))
}

type dialogPicker struct{}

func (dialogPicker) PickInputImage(parent fyne.Window, onPicked func(path string, err error)) {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			onPicked("", err)
			return
		}
		if reader == nil {
			onPicked("", nil)
			return
		}
		defer reader.Close()
		onPicked(filepath.Clean(reader.URI().Path()), nil)
	}, parent)
	fileDialog.SetTitleText("Select Input Image")
	fileDialog.SetFilter(storage.NewExtensionFileFilter(supportedImageExtensions))
	fileDialog.Show()
}

func (dialogPicker) PickOutputDir(parent fyne.Window, onPicked func(path string, err error)) {
	folderDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			onPicked("", err)
			return
		}
		if uri == nil {
			onPicked("", nil)
			return
		}
		if uri.Path() == "" {
			onPicked("", fmt.Errorf("selected folder did not provide a local path"))
			return
		}
		onPicked(filepath.Clean(uri.Path()), nil)
	}, parent)
	folderDialog.SetTitleText("Select Output Folder")
	folderDialog.Show()
}
