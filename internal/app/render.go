package app

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"godocmirrortranslator/internal/provider"
	base "godocmirrortranslator/internal/renderer"
)

const DefaultOutputTemplate = "{input_basename}_{provider}_{timestamp}.svg"

var atomicWriteFile = writeAtomically

type RenderRequest struct {
	InputPath      string
	OutputDir      string
	OutputTemplate string
	ProviderName   string
	ProviderConfig provider.ProviderConfig
	RendererName   string
	Model          string
	SourceLanguage string
	TargetLanguage string
	Timeout        time.Duration
	SaveLayoutJSON bool
	RenderOptions  base.RenderOptions
	Now            func() time.Time
}

type RenderResult struct {
	OutputPath     string
	LayoutJSONPath string
}

func (a *Application) RunRender(ctx context.Context, stdout io.Writer, req RenderRequest) error {
	result, err := a.Render(ctx, req)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, result.OutputPath)
	return err
}

func (a *Application) ValidateInputImage(path string) error {
	_, _, err := validateInputImage(path)
	return err
}

func (a *Application) Render(ctx context.Context, req RenderRequest) (RenderResult, error) {
	if req.InputPath == "" {
		return RenderResult{}, fmt.Errorf("input path is required")
	}
	if req.ProviderName == "" {
		req.ProviderName = "mock"
	}
	if req.RendererName == "" {
		req.RendererName = "svg"
	}
	if req.SourceLanguage == "" {
		req.SourceLanguage = "Ukrainian"
	}
	if req.TargetLanguage == "" {
		req.TargetLanguage = "German"
	}
	if req.OutputTemplate == "" {
		req.OutputTemplate = DefaultOutputTemplate
	}
	if req.Now == nil {
		req.Now = time.Now
	}
	if req.Timeout <= 0 {
		req.Timeout = 30 * time.Second
	}
	if req.OutputDir == "" {
		req.OutputDir = filepath.Dir(req.InputPath)
	}

	providerFactory, ok := a.ProviderFactories[req.ProviderName]
	if !ok {
		return RenderResult{}, fmt.Errorf("unknown provider %q", req.ProviderName)
	}
	providerImpl := providerFactory(req.ProviderConfig)
	if err := providerImpl.ValidateConfig(req.ProviderConfig); err != nil {
		return RenderResult{}, fmt.Errorf("validate provider config: %w", err)
	}
	rendererImpl, ok := a.Renderers[req.RendererName]
	if !ok {
		return RenderResult{}, fmt.Errorf("unknown renderer %q", req.RendererName)
	}

	imageWidth, imageHeight, err := validateInputImage(req.InputPath)
	if err != nil {
		return RenderResult{}, err
	}

	renderCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	page, err := providerImpl.AnalyzePage(renderCtx, provider.AnalyzeRequest{
		ImagePath:         req.InputPath,
		SourceImageWidth:  imageWidth,
		SourceImageHeight: imageHeight,
		SourceLanguage:    req.SourceLanguage,
		TargetLanguage:    req.TargetLanguage,
		Model:             req.Model,
		Timeout:           req.Timeout,
	})
	if err != nil {
		return RenderResult{}, fmt.Errorf("analyze page: %w", err)
	}
	page.Normalize()
	if err := page.Validate(); err != nil {
		return RenderResult{}, fmt.Errorf("validate provider output: %w", err)
	}

	renderBytes, err := rendererImpl.Render(renderCtx, page, req.RenderOptions)
	if err != nil {
		return RenderResult{}, fmt.Errorf("render output: %w", err)
	}

	outputName := outputFileName(req.OutputTemplate, req.ProviderName, req.Model, req.InputPath, req.Now(), rendererImpl.FileExtension())
	outputPath, err := availablePath(filepath.Join(req.OutputDir, outputName))
	if err != nil {
		return RenderResult{}, fmt.Errorf("choose output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return RenderResult{}, fmt.Errorf("create output directory: %w", err)
	}
	if err := atomicWriteFile(outputPath, renderBytes); err != nil {
		return RenderResult{}, fmt.Errorf("write svg: %w", err)
	}

	result := RenderResult{OutputPath: outputPath}
	if req.SaveLayoutJSON {
		jsonPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".json"
		jsonBytes, err := json.MarshalIndent(page, "", "  ")
		if err != nil {
			return RenderResult{}, cleanupRenderOutputs([]string{outputPath}, fmt.Errorf("marshal layout json: %w", err))
		}
		if err := atomicWriteFile(jsonPath, append(jsonBytes, '\n')); err != nil {
			return RenderResult{}, cleanupRenderOutputs([]string{outputPath, jsonPath}, fmt.Errorf("write layout json: %w", err))
		}
		result.LayoutJSONPath = jsonPath
	}

	return result, nil
}

func validateInputImage(path string) (int, int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, fmt.Errorf("stat input image: %w", err)
	}
	if info.Size() > 20*1024*1024 {
		return 0, 0, fmt.Errorf("input image exceeds 20 MiB")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png":
		file, err := os.Open(path)
		if err != nil {
			return 0, 0, fmt.Errorf("open input image: %w", err)
		}
		defer file.Close()

		cfg, _, err := image.DecodeConfig(file)
		if err != nil {
			return 0, 0, fmt.Errorf("decode input image config: %w", err)
		}
		return cfg.Width, cfg.Height, nil
	case ".webp":
		data, err := os.ReadFile(path)
		if err != nil {
			return 0, 0, fmt.Errorf("read input image: %w", err)
		}
		width, height, err := decodeWebPDimensions(data)
		if err != nil {
			return 0, 0, fmt.Errorf("decode webp dimensions: %w", err)
		}
		return width, height, nil
	default:
		return 0, 0, fmt.Errorf("unsupported image format %q", ext)
	}
}

func decodeWebPDimensions(data []byte) (int, int, error) {
	if len(data) < 30 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, fmt.Errorf("invalid webp header")
	}
	chunk := string(data[12:16])
	switch chunk {
	case "VP8X":
		if len(data) < 30 {
			return 0, 0, fmt.Errorf("truncated VP8X header")
		}
		widthMinusOne := int(data[24]) | int(data[25])<<8 | int(data[26])<<16
		heightMinusOne := int(data[27]) | int(data[28])<<8 | int(data[29])<<16
		return widthMinusOne + 1, heightMinusOne + 1, nil
	case "VP8L":
		if len(data) < 25 {
			return 0, 0, fmt.Errorf("truncated VP8L header")
		}
		bits := binary.LittleEndian.Uint32(data[21:25])
		width := int(bits&0x3FFF) + 1
		height := int((bits>>14)&0x3FFF) + 1
		return width, height, nil
	case "VP8 ":
		if len(data) < 30 {
			return 0, 0, fmt.Errorf("truncated VP8 header")
		}
		width := int(binary.LittleEndian.Uint16(data[26:28]) & 0x3FFF)
		height := int(binary.LittleEndian.Uint16(data[28:30]) & 0x3FFF)
		if width == 0 || height == 0 {
			return 0, 0, fmt.Errorf("invalid VP8 dimensions")
		}
		return width, height, nil
	default:
		return 0, 0, fmt.Errorf("unsupported webp chunk %q", chunk)
	}
}

func outputFileName(template, providerName, model, inputPath string, now time.Time, extension string) string {
	name := template
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	replacements := map[string]string{
		"{input_basename}": baseName,
		"{provider}":       providerName,
		"{model}":          model,
		"{date}":           now.Format("2006-01-02"),
		"{time}":           now.Format("15-04-05"),
		"{timestamp}":      now.Format("20060102-150405"),
	}
	for key, value := range replacements {
		name = strings.ReplaceAll(name, key, value)
	}
	if filepath.Ext(name) == "" {
		name += extension
	}
	return name
}

func availablePath(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	} else if err != nil {
		return "", err
	}
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
}

func writeAtomically(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}

func cleanupRenderOutputs(paths []string, renderErr error) error {
	var cleanupFailures []string
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			cleanupFailures = append(cleanupFailures, fmt.Sprintf("%s: %v", path, err))
		}
	}
	if len(cleanupFailures) == 0 {
		return renderErr
	}
	return fmt.Errorf("%w; cleanup failed: %s", renderErr, strings.Join(cleanupFailures, "; "))
}
