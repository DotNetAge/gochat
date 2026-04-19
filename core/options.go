package core

// Options holds per-request parameters
type Options struct {
	Model            string
	Temperature      *float64 // pointer so zero-value is distinguishable from "not set"
	MaxTokens        *int
	TopP             *float64
	Stop             []string
	Tools            []Tool
	SystemPrompt     string // prepended as system message if set
	Thinking         bool   // enables extended thinking/reasoning (provider-dependent)
	ThinkingBudget   int    // max tokens for thinking (0 = provider default)
	EnableSearch     bool   // Qwen/compatible-mode search
	IncrementalOutput bool   // stream incremental output (DeepSeek, Qwen)
	Format           string // response format, e.g., "json" (Ollama)
	KeepAlive         string // keep model in memory duration, e.g., "5m" (Ollama)
	UsageCallback     func(Usage)
	Attachments       []Attachment
	TopK              *int
	PresencePenalty   *float64
	FrequencyPenalty  *float64
	ParallelToolCalls *bool
	ToolChoice        interface{}
	ResponseFormat    string
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

// WithIncrementalOutput enables incremental output for streaming (DeepSeek, Qwen)
func WithIncrementalOutput(enabled bool) Option {
	return func(o *Options) {
		o.IncrementalOutput = enabled
	}
}

// WithFormat sets the response format (e.g., "json" for Ollama)
func WithFormat(format string) Option {
	return func(o *Options) {
		o.Format = format
	}
}

// WithKeepAlive sets the duration to keep the model in memory (Ollama)
func WithKeepAlive(duration string) Option {
	return func(o *Options) {
		o.KeepAlive = duration
	}
}

// WithAttachments attaches files or media to the message context
func WithAttachments(attachments ...Attachment) Option {
	return func(o *Options) {
		o.Attachments = append(o.Attachments, attachments...)
	}
}

// WithUsageCallback sets a callback to be called when usage info is available
func WithUsageCallback(fn func(Usage)) Option {
	return func(o *Options) { o.UsageCallback = fn }
}

// WithTopK sets the top-k sampling parameter
func WithTopK(k int) Option {
	return func(o *Options) { o.TopK = &k }
}

// WithPresencePenalty sets the presence penalty parameter
func WithPresencePenalty(p float64) Option {
	return func(o *Options) { o.PresencePenalty = &p }
}

// WithFrequencyPenalty sets the frequency penalty parameter
func WithFrequencyPenalty(p float64) Option {
	return func(o *Options) { o.FrequencyPenalty = &p }
}

// WithParallelToolCalls sets whether to allow parallel tool calls
func WithParallelToolCalls(parallel bool) Option {
	return func(o *Options) { o.ParallelToolCalls = &parallel }
}

// WithToolChoice sets the tool choice parameter
func WithToolChoice(choice interface{}) Option {
	return func(o *Options) { o.ToolChoice = choice }
}

// WithResponseFormat sets the response format (e.g., "json" or "text")
func WithResponseFormat(format string) Option {
	return func(o *Options) { o.ResponseFormat = format }
}

// ApplyOptions builds Options from a list of Option funcs
func ApplyOptions(opts ...Option) Options {
	var o Options
	for _, fn := range opts {
		fn(&o)
	}
	return o
}
