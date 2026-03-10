# Awesome Repositories Submission Strategy

Submitting GoChat to popular `awesome-*` lists on GitHub is one of the highest ROI (Return on Investment) marketing strategies for an open-source project. These lists have immense SEO value and are the first place developers look when searching for new tools.

Here is the targeted list of repositories and the exact markdown to submit via Pull Request.

## 1. awesome-go
**Repo**: [avelino/awesome-go](https://github.com/avelino/awesome-go)
**Category**: `Machine Learning` or `Utilities`
**PR Markdown**:
```markdown
* [gochat](https://github.com/DotNetAge/gochat) - A unified, type-safe LLM client SDK for Go. Supports seamless switching between OpenAI, Claude, DeepSeek, Qwen, and Ollama, with native features for Deep Thinking (Reasoning), Smart Secret Hosting, and enterprise OAuth2/Device Code AuthManager.
```
*Note: awesome-go has strict requirements (e.g., must have tests, Go Report Card, valid license, etc.). Ensure your repo meets their contributing guidelines before submitting.*

## 2. awesome-llm
**Repo**: [Hannibal046/Awesome-LLM](https://github.com/Hannibal046/Awesome-LLM)
**Category**: `Tools & Frameworks` or `API Wrappers`
**PR Markdown**:
```markdown
* [GoChat](https://github.com/DotNetAge/gochat) - An enterprise-ready Go SDK that unifies OpenAI, Claude, DeepSeek, and local Ollama models under one clean interface. It features native reasoning-chain parsing, built-in web search, and an advanced AuthManager for enterprise OAuth gateways.
```

## 3. awesome-generative-ai
**Repo**: [steven2358/awesome-generative-ai](https://github.com/steven2358/awesome-generative-ai)
**Category**: `Development Tools` -> `Libraries & Frameworks`
**PR Markdown**:
```markdown
* [GoChat](https://github.com/DotNetAge/gochat) - The ultimate Go client for LLMs. It smooths out API differences across major providers (OpenAI, Anthropic, Alibaba Qwen, DeepSeek) and handles complex state like multi-turn contexts, tool calling, and OAuth token refreshing out-of-the-box.
```

## Submission Workflow (How to do it)
1. **Fork** the target repository.
2. **Clone** your fork locally.
3. Read their `CONTRIBUTING.md` carefully (crucial for `awesome-go`).
4. Edit the `README.md` to insert your project in the appropriate alphabetical order within the chosen section.
5. Commit and push: `git commit -m "Add gochat to Machine Learning section"`
6. Open a Pull Request from your fork to the original repo. Keep the PR description polite and brief.
