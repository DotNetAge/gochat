package core

// Options holds per-request parameters
type Options struct {
	Model          string
	Temperature    *float64 // pointer so zero-value is distinguishable from "not set"
	MaxTokens      *int
	TopP           *float64
	Stop           []string
	Tools          []Tool
	SystemPrompt   string // prepended as system message if set
	Thinking       bool   // enables extended thinking/reasoning (provider-dependent)
	ThinkingBudget int    // max tokens for thinking (0 = provider default)
	EnableSearch   bool     // Qwen/compatible-mode search
	UsageCallback  func(Usage)
}

// Option is a functional option for Chat/ChatStream
type Option func(*Options)

// WithModel sets the model to use for this request
func WithModel(model string) Option {
	return func(o *Options) { o.Model = model }
}

// WithTemperature sets the temperature for generation
func WithTemperature(t float64) Option {
	return func(o *Options) { o.Temperature = &t }
}

// WithMaxTokens sets the maximum tokens to generate
func WithMaxTokens(n int) Option {
	return func(o *Options) { o.MaxTokens = &n }
}

// WithTopP sets the top-p sampling parameter
func WithTopP(p float64) Option {
	return func(o *Options) { o.TopP = &p }
}

// WithStop sets stop sequences
func WithStop(stops ...string) Option {
	return func(o *Options) { o.Stop = stops }
}

// WithTools sets available tools for the model to call
func WithTools(tools ...Tool) Option {
	return func(o *Options) { o.Tools = tools }
}

// WithSystemPrompt sets a system prompt (prepended as system message)
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) { o.SystemPrompt = prompt }
}

// WithThinking enables extended thinking/reasoning with optional token budget
func WithThinking(budget int) Option {
	return func(o *Options) {
		o.Thinking = true
		o.ThinkingBudget = budget
	}
}

// WithEnableSearch enables web search for models that support it (e.g. Qwen)
func WithEnableSearch(enabled bool) Option {
	return func(o *Options) {
		o.EnableSearch = enabled
	}
}

// WithUsageCallback sets a callback to be called when usage info is available
func WithUsageCallback(fn func(Usage)) Option {
	return func(o *Options) { o.UsageCallback = fn }
}

// ApplyOptions builds Options from a list of Option funcs
func ApplyOptions(opts ...Option) Options {
	var o Options
	for _, fn := range opts {
		fn(&o)
	}
	return o
}
