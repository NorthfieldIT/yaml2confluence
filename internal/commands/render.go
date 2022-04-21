package commands

import (
	"fmt"

	"github.com/NorthfieldIT/yaml2confluence/internal/cli"
	"github.com/NorthfieldIT/yaml2confluence/internal/services"
	"github.com/docopt/docopt-go"
)

type RenderCmd struct {
	service services.IRenderSrv
}

func (RenderCmd) Usage() string {
	return `
Usage:
	y2c render <file> [-o <format> | --output <format>]

Options:
	<file>     							The YAML resource to render
	-o <format>, --output <format>    	The phase to render to (yaml,json,mst)
`
}

func (rc RenderCmd) Handler(args docopt.Opts) {
	markup := rc.service.RenderSingleResource(args["<file>"].(string))
	fmt.Println(markup)
}

func init() {
	cli.RegisterCommand("render", RenderCmd{services.NewRenderService()})
}
