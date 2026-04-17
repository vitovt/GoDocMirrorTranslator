//go:build (linux && !android) || (darwin && !ios) || windows

package gui

import (
	"errors"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	nativedialog "github.com/sqweek/dialog"
)

type fallbackPickerStub struct {
	inputCalled  bool
	outputCalled bool
	inputPath    string
	outputPath   string
	inputErr     error
	outputErr    error
}

func (p *fallbackPickerStub) PickInputImage(_ fyne.Window, onPicked func(path string, err error)) {
	p.inputCalled = true
	onPicked(p.inputPath, p.inputErr)
}

func (p *fallbackPickerStub) PickOutputDir(_ fyne.Window, onPicked func(path string, err error)) {
	p.outputCalled = true
	onPicked(p.outputPath, p.outputErr)
}

func TestNativeDesktopPickerPickInputImageUsesNativePath(t *testing.T) {
	fallback := &fallbackPickerStub{}
	picker := nativeDesktopPicker{
		fallback: fallback,
		pickInput: func() (string, error) {
			return filepath.Join(string(filepath.Separator), "tmp", "picked", "..", "page.png"), nil
		},
	}

	var gotPath string
	var gotErr error
	picker.PickInputImage(nil, func(path string, err error) {
		gotPath = path
		gotErr = err
	})

	if gotErr != nil {
		t.Fatalf("PickInputImage() error = %v", gotErr)
	}
	wantPath := filepath.Join(string(filepath.Separator), "tmp", "page.png")
	if gotPath != wantPath {
		t.Fatalf("PickInputImage() path = %q, want %q", gotPath, wantPath)
	}
	if fallback.inputCalled {
		t.Fatal("PickInputImage() unexpectedly used fallback picker")
	}
}

func TestNativeDesktopPickerPickInputImageTreatsCancelAsNoSelection(t *testing.T) {
	fallback := &fallbackPickerStub{}
	picker := nativeDesktopPicker{
		fallback: fallback,
		pickInput: func() (string, error) {
			return "", nativedialog.ErrCancelled
		},
	}

	var gotPath string
	var gotErr error
	picker.PickInputImage(nil, func(path string, err error) {
		gotPath = path
		gotErr = err
	})

	if gotErr != nil {
		t.Fatalf("PickInputImage() error = %v, want nil", gotErr)
	}
	if gotPath != "" {
		t.Fatalf("PickInputImage() path = %q, want empty path", gotPath)
	}
	if fallback.inputCalled {
		t.Fatal("PickInputImage() unexpectedly used fallback picker on cancel")
	}
}

func TestNativeDesktopPickerPickOutputDirFallsBackAfterNativeError(t *testing.T) {
	fallback := &fallbackPickerStub{outputPath: filepath.Join(string(filepath.Separator), "tmp", "out")}
	picker := nativeDesktopPicker{
		fallback: fallback,
		pickOutput: func() (string, error) {
			return "", errors.New("native directory picker unavailable")
		},
	}

	var gotPath string
	var gotErr error
	picker.PickOutputDir(nil, func(path string, err error) {
		gotPath = path
		gotErr = err
	})

	if gotErr != nil {
		t.Fatalf("PickOutputDir() error = %v", gotErr)
	}
	if gotPath != fallback.outputPath {
		t.Fatalf("PickOutputDir() path = %q, want fallback path %q", gotPath, fallback.outputPath)
	}
	if !fallback.outputCalled {
		t.Fatal("PickOutputDir() did not use fallback picker after native error")
	}
}
