package resources

import (
	"errors"
	"fmt"
)

type TemplateProcessor struct {
	templates map[string]Template
}

type Template struct {
	Asset IAsset
	Data  string
}

func NewTemplateProcessor(templatesDir string) *TemplateProcessor {
	tp := TemplateProcessor{
		templates: loadAllTemplates(templatesDir),
	}

	return &tp
}

func (tp TemplateProcessor) Get(kind string) (string, error) {
	template, exists := tp.templates[kind]

	if !exists {
		return "", errors.New(fmt.Sprintf("No template exists for kind '%s'\n", kind))
	}

	if template.Data == "" {
		template.Data = template.Asset.ReadString()
	}

	return template.Data, nil
}

func (tp TemplateProcessor) GetAll() []Template {
	templates := []Template{}
	for _, t := range tp.templates {
		templates = append(templates, t)
	}

	return templates
}

func loadAllTemplates(templatesDir string) map[string]Template {
	templates := map[string]Template{}

	assets := append(GetBuiltinTemplates(), LoadAssets(templatesDir, []string{".mst", ".mustache"}, false)...)

	for _, asset := range assets {
		templates[asset.GetName()] = Template{Asset: asset}
	}

	return templates
}
