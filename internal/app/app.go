package app

import (
	"fmt"
	"sort"

	"godocmirrortranslator/internal/provider"
	"godocmirrortranslator/internal/provider/gemini"
	"godocmirrortranslator/internal/provider/mock"
	"godocmirrortranslator/internal/provider/openai"
	base "godocmirrortranslator/internal/renderer"
	fodgrenderer "godocmirrortranslator/internal/renderer/fodg"
	svgrenderer "godocmirrortranslator/internal/renderer/svg"
)

type ProviderFactory func(cfg provider.ProviderConfig) provider.Provider

type Application struct {
	ProviderFactories map[string]ProviderFactory
	Renderers         map[string]base.Renderer
	Version           string
}

func New(version string) *Application {
	providerFactories := map[string]ProviderFactory{
		"mock": func(cfg provider.ProviderConfig) provider.Provider {
			return mock.New(cfg)
		},
		"openai": func(cfg provider.ProviderConfig) provider.Provider {
			return openai.New(cfg)
		},
		"gemini": func(cfg provider.ProviderConfig) provider.Provider {
			return gemini.New(cfg)
		},
	}

	renderers := map[string]base.Renderer{}
	for _, current := range []base.Renderer{svgrenderer.New(), fodgrenderer.New()} {
		renderers[current.Name()] = current
	}

	return &Application{
		ProviderFactories: providerFactories,
		Renderers:         renderers,
		Version:           version,
	}
}

func (a *Application) ProviderNames() []string {
	names := make([]string, 0, len(a.ProviderFactories))
	for name := range a.ProviderFactories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (a *Application) SupportedModels(providerName string, cfg provider.ProviderConfig) []string {
	factory, ok := a.ProviderFactories[providerName]
	if !ok {
		return nil
	}
	models := append([]string(nil), factory(cfg).SupportedModels()...)
	sort.Strings(models)
	return models
}

func (a *Application) RendererNames() []string {
	names := make([]string, 0, len(a.Renderers))
	for name := range a.Renderers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (a *Application) ValidateProviderConfig(providerName string, cfg provider.ProviderConfig) error {
	factory, ok := a.ProviderFactories[providerName]
	if !ok {
		return fmt.Errorf("unknown provider %q", providerName)
	}
	return factory(cfg).ValidateConfig(cfg)
}
