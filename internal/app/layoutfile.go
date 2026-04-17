package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"godocmirrortranslator/internal/domain"
)

const savedLayoutSchemaVersion = 1

type savedLayoutFile struct {
	SchemaVersion int                 `json:"schema_version"`
	Page          domain.DocumentPage `json:"page"`
}

func newSavedLayoutFile(path string, page *domain.DocumentPage) (savedLayoutFile, error) {
	if page == nil {
		return savedLayoutFile{}, fmt.Errorf("page is required")
	}
	page.Normalize()
	if err := page.Validate(); err != nil {
		return savedLayoutFile{}, fmt.Errorf("validate page: %w", err)
	}

	cloned := cloneDocumentPage(*page)
	if sameDirectory(cloned.SourceImagePath, path) {
		cloned.SourceImagePath = filepath.Base(cloned.SourceImagePath)
	}

	return savedLayoutFile{
		SchemaVersion: savedLayoutSchemaVersion,
		Page:          cloned,
	}, nil
}

func writeSavedLayout(path string, page *domain.DocumentPage) error {
	layoutFile, err := newSavedLayoutFile(path, page)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(layoutFile, "", "  ")
	if err != nil {
		return fmt.Errorf("encode saved layout: %w", err)
	}
	return atomicWriteFile(path, append(encoded, '\n'))
}

func loadSavedLayout(path string) (*domain.DocumentPage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read layout json: %w", err)
	}

	var layoutFile savedLayoutFile
	if err := json.Unmarshal(data, &layoutFile); err != nil {
		return nil, fmt.Errorf("decode layout json: %w", err)
	}
	if layoutFile.SchemaVersion != savedLayoutSchemaVersion {
		return nil, fmt.Errorf("unsupported layout json schema version %d", layoutFile.SchemaVersion)
	}

	page := layoutFile.Page
	if page.SourceImagePath == "" {
		return nil, fmt.Errorf("layout json source image path is required")
	}
	if !filepath.IsAbs(page.SourceImagePath) {
		page.SourceImagePath = filepath.Join(filepath.Dir(path), page.SourceImagePath)
	}
	page.SourceImagePath = filepath.Clean(page.SourceImagePath)

	page.Normalize()
	if err := page.Validate(); err != nil {
		return nil, fmt.Errorf("validate layout json page: %w", err)
	}

	actualWidth, actualHeight, err := validateInputImage(page.SourceImagePath)
	if err != nil {
		return nil, fmt.Errorf("validate saved source image: %w", err)
	}
	if actualWidth != page.SourceImageWidth || actualHeight != page.SourceImageHeight {
		return nil, fmt.Errorf(
			"saved layout source image dimensions %dx%d do not match current source image %dx%d",
			page.SourceImageWidth,
			page.SourceImageHeight,
			actualWidth,
			actualHeight,
		)
	}

	return &page, nil
}

func cloneDocumentPage(page domain.DocumentPage) domain.DocumentPage {
	cloned := page
	cloned.Blocks = append([]domain.TextBlock(nil), page.Blocks...)
	if page.Metadata != nil {
		cloned.Metadata = make(map[string]string, len(page.Metadata))
		for key, value := range page.Metadata {
			cloned.Metadata[key] = value
		}
	}
	return cloned
}

func sameDirectory(a, b string) bool {
	aDir, aErr := filepath.Abs(filepath.Dir(a))
	bDir, bErr := filepath.Abs(filepath.Dir(b))
	if aErr == nil && bErr == nil {
		return aDir == bDir
	}
	return filepath.Clean(filepath.Dir(a)) == filepath.Clean(filepath.Dir(b))
}
