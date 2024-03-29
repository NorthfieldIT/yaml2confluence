package resources

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"gopkg.in/yaml.v3"
)

var SUPPORTED_TAGS map[string]bool = map[string]bool{
	"!!str":   true,
	"!!bool":  true,
	"!!int":   true,
	"!!float": true,
	"!!seq":   true,
	"!!map":   true,
}

type YqHooks struct {
	defaults  string
	overrides string
	merges    string
	while     string
	yq        []string
}

var encoder yqlib.Encoder = yqlib.NewJSONEncoder(0, false, false)
var evaluator yqlib.Evaluator = yqlib.NewAllAtOnceEvaluator()

func NewYqHook(defaults, overrides, merges yaml.Node, while string, yqCmds []string) (YqHooks, error) {
	mergeNodes := map[string]yaml.Node{
		"defaults":  defaults,
		"overrides": overrides,
		"merges":    merges,
	}
	mergeInstructions := map[string][]bool{
		"defaults":  {false, false},
		"overrides": {true, false},
		"merges":    {true, true},
	}
	commands := map[string]string{}

	for mType, node := range mergeNodes {
		cmd, err := nodeToYqCommand(node, mergeInstructions[mType][0], mergeInstructions[mType][1])
		if err != nil {
			return YqHooks{}, errors.New(fmt.Sprintf("Hook: %s\n\t%s", mType, err.Error()))
		}
		commands[mType] = cmd
	}

	yqHooks := YqHooks{
		defaults:  commands["defaults"],
		overrides: commands["overrides"],
		merges:    commands["merges"],
		while:     while,
		yq:        yqCmds,
	}

	return yqHooks, nil
}

func (yh YqHooks) Run(node *yaml.Node) (*yaml.Node, error) {
	newNode := node
	var err error

	commands := []string{yh.defaults, yh.overrides, yh.merges}
	if yh.while == "" {
		commands = append(commands, yh.yq...)
	}

	for _, command := range commands {
		newNode, err = runYqCommand(command, newNode)
		if err != nil {
			return nil, err
		}
	}

	if yh.while != "" {
	out:
		for i := 0; i < 10; i++ {
			if !whileCondition(yh.while, newNode) {
				break out
			}

			for _, command := range yh.yq {
				newNode, err = runYqCommand(command, newNode)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return newNode, nil
}

func whileCondition(command string, node *yaml.Node) bool {
	boolNode, err := runYqCommand(command, node)
	if err != nil {
		return false
	}

	if boolNode.Tag != "!!bool" {
		return false
	}

	return boolNode.Value == "true"
}

func runYqCommand(command string, node *yaml.Node) (*yaml.Node, error) {
	if command != "" {
		list, err := evaluator.EvaluateNodes(command, node)
		if err != nil {
			return nil, err
		}
		newNode := list.Front().Value.(*yqlib.CandidateNode).Node

		return newNode, nil
	}

	return node, nil
}

func nodeToYqCommand(node yaml.Node, overide bool, merge bool) (string, error) {
	setExpressions := []string{}
	for i := range node.Content {
		if i%2 == 0 { //even
			key := node.Content[i].Value
			val := node.Content[i+1].Value
			tag := node.Content[i+1].Tag

			if !isSupportedTag(tag) {
				return "", errors.New(fmt.Sprintf("Unsupported tag: %s", tag))
			}

			if tag == "!!seq" || tag == "!!map" {
				val = toJson(node.Content[i+1])
			}

			setExpressions = append(setExpressions, getSetExpression(key, val, tag))
		} else {
			continue
		}
	}

	if len(setExpressions) > 0 {
		mergeType := "n"
		if overide {
			mergeType = ""
		}
		if merge {
			mergeType += "+"
		}
		return fmt.Sprintf("{} as $d|%s|. *%s $d", strings.Join(setExpressions, "|"), mergeType), nil
	}

	return "", nil
}

func isSupportedTag(tag string) bool {
	supported, exists := SUPPORTED_TAGS[tag]

	return exists && supported
}

func getSetExpression(key, val, tag string) string {
	if tag == "!!str" {
		return fmt.Sprintf(`$d.%s="%s"`, key, val)
	}

	return fmt.Sprintf(`$d.%s=%s`, key, val)
}

func toJson(node *yaml.Node) string {
	var buf bytes.Buffer
	err := encoder.Encode(&buf, node)
	if err != nil {
		panic(err)
	}

	return buf.String()
}

// HOW TO GET *yaml.Node FROM yqlib.NewAllAtOnceEvaluator().EvaluateNodes
// 	fmt.Println(list.Front().Value.(*yqlib.CandidateNode).Node)
