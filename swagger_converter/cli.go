package main

import (
	"log"
	"os"

	"github.com/jaffee/commandeer"
)

// Converter struct will provision flags for this scripts
// see: github.com/jaffee/commandeer
type Converter struct {
	Url               string `help:"Url to the swagger to fetch"`
	SwaggerFile       string `help:"Path where to read or save the swagger file"`
	OpenApiSourceFile string `help:"Path where to generate the source OpenAPI v3"`
	OpenApiDestFile   string `help:"Where to put the converted OpenAPI file"`
	Clean             bool   `help:"if true, will re-download the swagger file from url, deleting the local one"`
	Exec              bool   `help:"run the conversion"`
}

// NewConverter returns the default values of the CLI
func NewConverter() *Converter {
	return &Converter{
		Url:               "https://docs.cycloid.io/api/swagger.yaml",
		SwaggerFile:       ".ci/swagger.yaml",
		OpenApiSourceFile: ".ci/openapi/openapi.yaml",
		OpenApiDestFile:   "./openapi.yaml",
		Clean:             false,
		Exec:              false,
	}
}

// Run is the function called by the CLI
func (c Converter) Run() error {
	if c.Clean {
		if err := os.Remove(c.SwaggerFile); err != nil {
			return err
		}
	}

	if err := c.Fetch(); err != nil {
		return err
	}

	if err := c.Convert(); err != nil {
		return err
	}

	return nil
}

func main() {
	converter := NewConverter()

	// Make the program display help by default
	if !converter.Exec {
		os.Args = append(os.Args, "-h")
	}

	if err := commandeer.Run(converter); err != nil {
		log.Fatal(err)
	}
}
