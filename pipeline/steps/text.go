package steps

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/DotNetAge/gochat/pipeline"
)

// TemplateStep is a pipeline step that renders a text template using data from the state.
type TemplateStep struct {
	templateStr string
	outputKey   string
	inputKeys   []string
}

// NewTemplateStep creates a step that renders a template.
// - templateStr: The Go text/template string.
// - outputKey: The state key to write the rendered string to.
// - inputKeys: Keys from the state to pass into the template as variables.
func NewTemplateStep(templateStr, outputKey string, inputKeys ...string) *TemplateStep {
	return &TemplateStep{
		templateStr: templateStr,
		outputKey:   outputKey,
		inputKeys:   inputKeys,
	}
}

// Name returns the step's name.
func (s *TemplateStep) Name() string {
	return "RenderTemplate"
}

// Execute renders the template.
func (s *TemplateStep) Execute(ctx context.Context, state *pipeline.State) error {
	tmpl, err := template.New("step").Parse(s.templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := make(map[string]interface{})
	for _, key := range s.inputKeys {
		if val, ok := state.Get(key); ok {
			data[key] = val
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	state.Set(s.outputKey, buf.String())
	return nil
}
