package resources

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	. "github.com/flant/libjq-go"
	"github.com/flant/libjq-go/pkg/jq"
	"github.com/mattn/go-zglob"
	"gopkg.in/yaml.v3"
)

type HookProcessor struct {
	shouldPrecompile bool
	hooks            map[string]*Hook
	kindHooks        map[string]*Hook
	patternHooks     []*Hook
	lsCache          *LsCache
}

type Hook struct {
	Asset  IAsset
	Config *HookConfig
}

type HookConfig struct {
	Target    string    `yaml:"target"`
	Priority  int       `yaml:"priority"`
	ListFiles ListFiles `yaml:"listFiles"`
	Defaults  yaml.Node `yaml:"defaults"`
	Overrides yaml.Node `yaml:"overrides"`
	Merges    yaml.Node `yaml:"merges"`
	YqWhile   string    `yaml:"yqWhile"`
	Yq        []string  `yaml:"yq"`
	Jq        []string  `yaml:"jq"`
	Header    string    `yaml:"header"`
	Footer    string    `yaml:"footer"`
}

type ListFiles struct {
	EnvVar string `yaml:"envVar"`
	Glob   string `yaml:"glob"`
}

func (lf ListFiles) isValid() bool {
	return lf.EnvVar != "" && lf.Glob != ""
}

type HookSet struct {
	Jq     []JqCommand
	Yq     []YqHooks
	Ls     Ls
	Header string
	Footer string
}
type LsCache struct {
	store map[ListFiles]string
}

func (c *LsCache) Get(lf ListFiles) (string, bool) {
	val, exists := c.store[lf]

	return val, exists
}

func (c *LsCache) Set(lf ListFiles, data string) {
	c.store[lf] = data
}

type Ls struct {
	config ListFiles
	cache  *LsCache
}

func (ls *Ls) Run() {
	if !ls.config.isValid() {
		return
	}

	if data, exists := ls.cache.Get(ls.config); exists {
		os.Setenv(ls.config.EnvVar, data)
		return
	}

	data := GlobYaml(ls.config.Glob)
	ls.cache.Set(ls.config, data)
	os.Setenv(ls.config.EnvVar, data)
}

type JqCommand struct {
	precompiled *jq.JqProgram
	Cmd         string
	Hook        *Hook
}

func (jc *JqCommand) precompile() error {
	prg, err := Jq().Program(jc.Cmd).Precompile()
	if err != nil {
		return err
	}

	jc.precompiled = prg

	return nil
}

func (jc *JqCommand) Run(json string) (string, error) {
	if jc.precompiled != nil {
		return jc.precompiled.Run(json)
	} else {
		return Jq().Program(jc.Cmd).Run(json)
	}
}

func NewHookProcessor(hooksDir string, precompile bool) *HookProcessor {
	hp := HookProcessor{
		shouldPrecompile: precompile,
		hooks:            map[string]*Hook{},
		kindHooks:        map[string]*Hook{},
		lsCache: &LsCache{
			store: map[ListFiles]string{},
		},
	}

	hooks := append(loadHooks(hooksDir))

	for _, hook := range hooks {
		hp.hooks[hook.Asset.GetName()] = hook
		if hook.Config.Target == "" {
			hp.kindHooks[hook.Asset.GetName()] = hook
		} else {
			hp.patternHooks = append(hp.patternHooks, hook)
		}
	}

	return &hp
}

func (hp *HookProcessor) Get(hookName string) *Hook {
	return hp.hooks[hookName]
}

func (hp *HookProcessor) GetHooks(kind string) []*Hook {
	hooks := []*Hook{}

	if kindHook, exists := hp.kindHooks[kind]; exists {
		hooks = append(hooks, kindHook)
	}

	for _, ph := range hp.patternHooks {
		if matched, _ := regexp.MatchString(ph.Config.Target, kind); matched {
			hooks = append(hooks, ph)
		}
	}

	sort.SliceStable(hooks, func(i, j int) bool {
		return hooks[i].Config.Priority < hooks[j].Config.Priority
	})

	return hooks
}
func (hp *HookProcessor) GetAll() []*Hook {
	hooks := append([]*Hook{}, hp.patternHooks...)

	for _, h := range hp.kindHooks {
		hooks = append(hooks, h)
	}

	return hooks
}

func (hp *HookProcessor) GetHookSet(kind string) HookSet {
	hookset := HookSet{}
	headers := []string{}
	footers := []string{}

	for _, hook := range hp.GetHooks(kind) {
		hookset.Ls = Ls{
			config: hook.Config.ListFiles,
			cache:  hp.lsCache,
		}

		for _, jq := range hook.Config.Jq {
			jqCommand := JqCommand{Cmd: jq, Hook: hook}
			if hp.shouldPrecompile {
				err := jqCommand.precompile()
				if err != nil {
					fmt.Printf("Failed to precompile jq statement\nHook name: %s\nFile: %s\njq: %s\nError: %s", hook.Asset.GetName(), hook.Asset.GetPath(), jq, err.Error())
					os.Exit(1)
				}
			}
			hookset.Jq = append(hookset.Jq, jqCommand)
		}

		yqHooks, err := NewYqHook(hook.Config.Defaults, hook.Config.Overrides, hook.Config.Merges, hook.Config.YqWhile, hook.Config.Yq)
		if err != nil {
			panic(err)
		}

		hookset.Yq = append(hookset.Yq, yqHooks)

		if hook.Config.Header != "" {
			headers = append(headers, hook.Config.Header)
		}
		if hook.Config.Footer != "" {
			footers = append(footers, hook.Config.Footer)
		}
	}

	hookset.Header += strings.Join(headers, "\n")
	hookset.Footer += strings.Join(footers, "\n")

	return hookset
}

func loadHooks(hooksDir string) []*Hook {
	hooks := []*Hook{}

	assets := append(GetBuiltinHooks(), LoadAssets(hooksDir, []string{".yml", ".yaml"}, true)...)

	for _, asset := range assets {
		config, err := loadHookConfig(asset.ReadBytes())
		if err != nil {
			panic(err)
		}

		hook := Hook{
			Asset:  asset,
			Config: config,
		}

		hooks = append(hooks, &hook)
	}

	return hooks
}

func loadHookConfig(data []byte) (*HookConfig, error) {
	hookConfig := HookConfig{}

	node := yaml.Node{}
	yaml.Unmarshal(data, &node)

	if len(node.Content) != 1 || node.Content[0].ShortTag() != "!!map" {
		return nil, errors.New("Invalid hook yaml")
	}

	ensureArray("yq", &node)
	ensureArray("jq", &node)

	err := node.Decode(&hookConfig)
	if err != nil {
		return nil, err
	}

	return &hookConfig, nil
}

/*
allows hooks to be defined as single value or an array

jq: .user as $s | .user |= "mike"

# BECOMES

jq:
  - .user as $s | .user |= "mike"
*/
func ensureArray(rootKey string, node *yaml.Node) {
	content := node.Content[0].Content
	for i := range content {
		if content[i].Value == rootKey && content[i+1].ShortTag() == "!!str" {
			seq := yaml.Node{
				Kind:    yaml.SequenceNode,
				Content: append([]*yaml.Node{}, content[i+1]),
			}
			content[i+1] = &seq
			break
		}
	}
}

func GlobYaml(glob string) string {
	spaceDir := os.Getenv("SPACE_DIR")
	matches, err := zglob.Glob(filepath.Join(spaceDir, glob))
	if err != nil {
		panic(err)
	}

	yamlLines := []string{}
	for _, m := range matches {
		path, err := filepath.Rel(spaceDir, m)
		if err != nil {
			panic(err)
		}
		yamlLines = append(yamlLines, fmt.Sprintf(" - %s", path))
	}

	return strings.Join(yamlLines, "\n")
}
