package resources

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aybabtme/orderedjson"
	"github.com/mattn/go-colorable"
	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"github.com/nwidger/jsoncolor"
	"gopkg.in/op/go-logging.v1"
	"gopkg.in/yaml.v3"
)

var yamlEncoder yqlib.Encoder
var prettyJsonEncoder yqlib.Encoder

func init() {
	// disable yqlib debug logging
	leveled := logging.AddModuleLevel(logging.NewLogBackend(os.Stderr, "", 0))
	leveled.SetLevel(logging.ERROR, "")
	yqlib.GetLogger().SetBackend(leveled)

	prettyJsonEncoder = yqlib.NewJSONEncoder(4, false, false)
	// YamlPrinter = yqlib.NewPrinter(yqlib.NewYamlEncoder(indent, colorsEnabled, printDocSeparators, unwrapScalar), yqlib.NewSinglePrinterWriter(writer))
}

func PrettyPrint(target RenderTarget, page *Page, w *os.File) {
	switch target {
	case YAML:
		PrettyPrintYaml(page.Resource.Node, w)
	case JSON:
		prettyPrintJson(page.Resource.ToOrderedMap(), w)
	case MST:
		fmt.Fprintln(w, page.Content.Markup)
	}
}

func PrettyPrintYaml(node *yaml.Node, w io.Writer) {
	prefs := yqlib.NewDefaultYamlPreferences()
	prefs.UnwrapScalar = false

	printer := yqlib.NewPrinter(yqlib.NewYamlEncoder(4, shouldColorize(), prefs), yqlib.NewSinglePrinterWriter(w))

	list, err := yqlib.NewAllAtOnceEvaluator().EvaluateNodes(".", node)
	if err != nil {
		panic(err)
	}
	printer.PrintResults(list)
}

type Encoder interface {
	SetIndent(string, string)
	Encode(interface{}) error
}

func prettyPrintJson(obj orderedjson.Map, w *os.File) {
	var enc Encoder = json.NewEncoder(w)
	if shouldColorize() {
		out := colorable.NewColorable(w) // needed for Windows
		enc = jsoncolor.NewEncoder(out)
	}

	enc.SetIndent("", "  ")
	err := enc.Encode(obj)
	if err != nil {
		panic(err)
	}
}

func shouldColorize() bool {
	colorsEnabled := false
	fileInfo, _ := os.Stdout.Stat()

	if (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		colorsEnabled = true
	}

	return colorsEnabled
}
