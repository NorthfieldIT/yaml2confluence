package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	"path/filepath"

	"github.com/NorthfieldIT/yaml2confluence/internal/utils"
	"github.com/cbroglie/mustache"
)

type RenderTarget uint32

const (
	YAML = 1 << iota
	JSON
	MST
)

type RenderTools struct {
	dirProps  utils.DirectoryProperties
	templates *TemplateProcessor
	hooks     *HookProcessor
	hasher    hash.Hash
}

func NewRenderTools(dirProps utils.DirectoryProperties, precompileJqHooks bool) *RenderTools {
	rt := RenderTools{
		dirProps:  dirProps,
		templates: NewTemplateProcessor(dirProps.TemplatesDir),
		hooks:     NewHookProcessor(dirProps.HooksDir, precompileJqHooks),
	}

	return &rt
}

// func (rt *RenderTools) GetTemplate(kind string) string {
// 	template, exists := rt.templates[kind]
// 	if !exists {
// 		template = loadTemplate(kind, rt.dirProps.TemplatesDir)
// 		rt.templates[kind] = template
// 	}

// 	return template
// }

func (rt *RenderTools) RenderTo(target RenderTarget, p *Page) {
	hookset := rt.hooks.GetHookSet(p.Resource.Kind)

	switch {
	case target >= YAML:
		for _, yq := range hookset.Yq {
			node, err := yq.Run(p.Resource.Node)
			if err != nil {
				panic(err)
			}

			p.Resource.Node = node
		}
		p.Resource.UpdateJson()
		fallthrough
	case target >= JSON:
		for _, jq := range hookset.Jq {
			res, err := jq.Run(p.Resource.Json)
			if err != nil {
				fmt.Printf("Failed to render %s\nError in hook: %s\n\njq %s\n%s\n\n", filepath.Join(rt.dirProps.SpaceDir, p.Resource.Path), jq.Hook.Asset.GetPath(), jq.Cmd, err.Error())
				os.Exit(1)
			}

			p.Resource.Json = res
		}
		p.Resource.UpdateKindAndTitle()
		fallthrough
	case target == MST:
		template, err := rt.templates.Get(p.Resource.Kind)
		if err != nil {
			fmt.Printf("Failed to render %s\n%s", filepath.Join(rt.dirProps.SpaceDir, p.Resource.Path), err.Error())
			os.Exit(1)
		}
		renderContent(p, template, hookset.Header, hookset.Footer)
	}
}

func (rt *RenderTools) RenderAll(pt *PageTree) {
	for _, page := range pt.GetPages() {
		rt.RenderTo(MST, page)
	}
}

func renderContent(p *Page, template string, header string, footer string) {
	// TODO handle error
	p.Content.Markup, _ = mustache.Render(template, p.Resource.ToObject())
	if header != "" {
		p.Content.Markup = header + "\n" + p.Content.Markup
	}
	if footer != "" {
		p.Content.Markup += "\n" + footer
	}
	hasher := sha256.New()
	hasher.Write([]byte(p.Content.Markup))
	p.Content.Sha256 = hex.EncodeToString(hasher.Sum(nil))
}
