package app

import (
	"sort"

	"godocmirrortranslator/internal/provider"
	"godocmirrortranslator/internal/provider/gemini"
	"godocmirrortranslator/internal/provider/mock"
	"godocmirrortranslator/internal/provider/openai"
	base "godocmirrortranslator/internal/renderer"
	svgrenderer "godocmirrortranslator/internal/renderer/svg"
)

type Application struct {
	Providers map[string]provider.Provider
	Renderers map[string]base.Renderer
	Version   string
}

func New(version string) *Application {
	providers := map[string]provider.Provider{}
	for _, current := range []provider.Provider{mock.New(), openai.New(), gemini.New()} {
		providers[current.Name()] = current
	}

	renderers := map[string]base.Renderer{}
	for _, current := range []base.Renderer{svgrenderer.New()} {
		renderers[current.Name()] = current
	}

	return &Application{
		Providers: providers,
		Renderers: renderers,
		Version:   version,
	}
}

func (a *Application) ProviderNames() []string {
	names := make([]string, 0, len(a.Providers))
	for name := range a.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
