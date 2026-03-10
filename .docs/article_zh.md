# 接了5家大模型API后，我被代码里的各种结构体逼疯了——于是有了 GoChat

如果你的业务系统最近正在尝试做“AI 转型”，或者是做个智能客服、知识库检索，那你一定经历过这样的痛点：

昨天老板说：“接入 OpenAI 的 GPT-4o 吧，效果最好。” 你去看了看文档，很快封装了一套请求逻辑。
今天老板说：“DeepSeek-R1 免费又好用，给我加上，还要把它的‘思考过程’在前端打印出来。”
明天安全部门可能又说：“公有云不安全，把咱们本地用 Ollama 跑的 Qwen 模型也接进来吧。”
后天运维大哥拿了个 Qwen Portal 企业专有网关过来：“别用 API Key，这是用 OAuth2 验证的。”

**结果就是，你的业务代码里充斥着大量的 `if-else`。**
你要写一堆转换器，把业务逻辑翻译成 OpenAI 格式、Anthropic 格式、本地格式……
你要在代码深处写死各种 `sk-xxx` 密钥。
每次增加一个大模型，你甚至要重写整个网络请求和流式解析（Streaming）引擎。

我们团队就是这样被折磨的。痛定思痛后，我们决定从零开始，写一个**真·生产环境可用**的 Go 语言大模型基建 SDK：[**GoChat**](https://github.com/DotNetAge/gochat)。

## 痛点一：每换一个模型，就要重写一次调用？

GoChat 最大的初衷就是**极简和统一**。无论你背后接的是 GPT-4、Claude 3.7 还是本地的 Ollama，在业务层看来，它只有一个核心接口：

```go
client.Chat(ctx, messages, core.WithTemperature(0.7))
```

想要无缝切模型？你只需要改一行初始化代码，不用去调整复杂的 Message Payload。所有的特性开关（比如开启内置工具、设定最大 Token、开启大模型内部联网搜索），统统通过 Go 语言最经典的 **Functional Options (`core.Option`)** 来控制，这极大程度保持了代码的清爽。

比如，你想让模型具备“联网搜索”超能力（以通义千问为例）：
```go
resp, _ := client.Chat(ctx, messages, core.WithEnableSearch(true))
```
就这么简单。

## 痛点二：API Key 满天飞，一不小心传到 GitHub？

很多开源的 Demo 喜欢直接把 API Key 写死在 `Config` 里，这在企业级开发中是大忌。为了解决这个问题，我们在 GoChat 底层内嵌了 **智能密钥托管 (Smart Secret Hosting)**。

你只需要像这样传一个“名字”给配置：
```go
client, _ := deepseek.New(deepseek.Config{
    Config: base.Config{
        APIKey: "DEEPSEEK_API_KEY", // 不是真实的 key，只是个代号
    },
})
```
引擎会在实际发起 HTTP 请求前，自动去系统环境变量中安全提取。你甚至不用拘泥于大小写，传个 `deepseek-api-key`，它也能自动转换为 `DEEPSEEK_API_KEY` 去找。代码里从此再无明文密钥。

## 痛点三：满血版模型的“深度思考 (Deep Thinking)”，到底怎么拿？

2026年了，各大模型的趋势是提供“思考链（Reasoning）”。比如 DeepSeek-R1 或者 o1，它们会先进行一大段内部推导，然后再输出最终答案。

很多老旧的 SDK 没有分离这两部分，导致你拿到的回答混杂在一起。在 GoChat 中，我们重新设计了流式事件 `StreamEvent`，不仅兼容标准的文本输出，更是**原生支持了各大厂思维链的拦截**。

```go
stream, _ := client.ChatStream(ctx, messages, core.WithThinking(0))

for stream.Next() {
    ev := stream.Event()
    if ev.Type == core.EventThinking { 
        fmt.Print("模型正在内心OS: ", ev.Content) 
    } else if ev.Type == core.EventContent {
        fmt.Print("最终回答: ", ev.Content)
    }
}
```
几行代码，就可以在你的终端或前端界面上，完美复刻出“正在思考”的进度条特效。

## 痛点四：企业级网关不仅要 Token，还要定期自动刷新？

其实大多数简单的 SDK 做到上面三点就结束了，但那依然不够“生产级”。

当你要对接大型企业的门户网关（比如 Qwen Portal 或 GCP Gemini）时，人家是不给你 API Key 的。人家只给你一组 OAuth Client ID，要求走标准的 Device Code Flow 或者回调验证。

这意味着什么？意味着你获取的 Token 会在 1 小时后过期！如果用普通 SDK，你必须自己在业务里写一套复杂的定时器和持久化逻辑。

这简直太痛苦了。因此，GoChat 直接给你搞了一个内置的 `AuthManager`。

```go
// 1. 注册提供商与 AuthManager
p := provider.NewQwenProvider()
authMgr := core.NewAuthManager(p, "token_store.json")

// 2. 无脑调 GetToken 就完事了
token, _ := authMgr.GetToken()

// 如果过期，它在底层偷偷给你 Refresh 拿新的
// 如果本地没有，它会自动弹授权链接让你点
```

不仅如此，由于考虑到不同企业业务部署差异，我们还提供了自定义 `TokenStore` 接口。你可以花 5 分钟写一个基于 Redis 的 Store，让你的微服务集群在获取 Token 时永远保持高可用和状态同步。

## 写在最后

造轮子不是目的，让业务开发人员**不被千奇百怪的 API 标准卡脖子**，才是 GoChat 想做的事。

无论你是独立开发者想快速接入多种免费模型，还是企业架构师正在寻找一个稳定、安全、原生支持 OAuth2 和多模态的 LLM 基建，都不妨来看看 GoChat。

**GitHub 开源地址**：[DotNetAge/gochat](https://github.com/DotNetAge/gochat)

欢迎提 Issue，欢迎来 Star！也祝各位在 AI 赋能业务的路上，少踩坑，多摸鱼！