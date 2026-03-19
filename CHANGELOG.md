
- v0.1.9 
1. 核心契约升级 (embedding.go)：
    - 新增了 MultimodalProvider 接口，继承原有的 Provider，并暴露了 EmbedImages(ctx, images [][]byte) 方法，让外层可以直接塞入图片二进制。
2. 纯 Go 图片预处理引擎 (image_processor.go)：
    - 新建了专用的 ImageProcessor 结构体。为了不破坏零依赖原则，我直接使用了标准库 image 和手动双线性插值实现了 Resize 和 CenterCrop（224x224）。
    - 内置了 CLIP 标准的 Mean [0.481, 0.457, 0.408] 和 Std [0.268, 0.261, 0.275] 归一化逻辑，最终转化为模型需要的四维 Float32 张量（[B, C, H, W]）。
3. 双塔模型架构组装 (clip.go)：
    - 新增了 CLIPProvider 的完整实现。内部封装了对 text_model.onnx 和 vision_model.onnx 的双重加载和调用流。
4. 工厂与下载器挂载 (factory.go / models.go / downloader.go)：
    - 增加常量 ModelTypeCLIP 自动识别规则。
    - 在 factory.go 暴露了专用的快速构建入口 WithCLIP(modelName, modelPath)。
    - downloader.go 已被我打入了官方推荐模型的配置。

> 关于中文语境的小提示：
> 这个标准 CLIP 对英文支持最完美。如果您的多模态系统绝大多数情况只被用户用来搜中文，您后续可以使用阿里的 OFA-Sys/chinese-clip-vit-base-patch16 （也可以在 HF 找到其 ONNX
  导出版本）。目前的代码和尺寸处理逻辑（224x224 分辨率）完全可以兼容它