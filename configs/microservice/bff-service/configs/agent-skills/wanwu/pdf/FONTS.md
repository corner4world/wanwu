# PDF 字体配置说明

## 概述

本 PDF skill 已配置智能字体选择功能，支持中英文混合文本的自动字体选择。**优先使用 CIDFont 方式**，无需字体文件，PDF 文件体积更小。

## 字体配置

### 主要字体（推荐）

| 字体名称 | 类型 | 用途 | 说明 |
|---------|------|------|------|
| **STSong-Light** | CIDFont | 中文文本 | **推荐**：PDF阅读器内置，无需字体文件 |
| **Times New Roman** | TTF | 英文/数字 | Liberation Serif 开源替代 |

### 备选字体（TTF）

| 字体名称 | 类型 | 用途 | 说明 |
|---------|------|------|------|
| Noto Serif CJK SC | TTF | 中文文本 | 宋体，需要字体文件 |
| Noto Sans CJK | TTF | 中文文本 | 黑体，需要字体文件 |
| WenQuanYi Zen Hei | TTF | 中文文本 | 文泉驿正黑，需要字体文件 |
| WenQuanYi Micro Hei | TTF | 中文文本 | 文泉驿微米黑，需要字体文件 |

## CIDFont vs TTF 对比

### CIDFont (推荐)

```python
from reportlab.pdfbase.cidfonts import UnicodeCIDFont
pdfmetrics.registerFont(UnicodeCIDFont('STSong-Light'))
```

**优势**：
- ✅ **无需字体文件** - PDF 阅读器内置支持
- ✅ **PDF 文件更小** - 不需要嵌入字体
- ✅ **完美兼容性** - Adobe 标准 CID 字体
- ✅ **完整字符集** - 支持所有中文字符
- ✅ **处理速度更快** - 无需加载字体文件

**劣势**：
- ⚠️ 仅支持标准字体（宋体、黑体等）
- ⚠️ 无法使用自定义字体

### TTF 字体

```python
from reportlab.pdfbase.ttfonts import TTFont
pdfmetrics.registerFont(TTFont('NotoSerifCJK', '/path/to/font.ttf'))
```

**优势**：
- ✅ 支持自定义字体
- ✅ 字体样式丰富

**劣势**：
- ❌ 需要安装字体文件
- ❌ PDF 文件体积大（字体嵌入）
- ❌ 依赖系统字体路径

### 字体选择规则

系统自动根据文本内容选择合适的字体：

- **中文文本** → STSong-Light (CIDFont)
- **英文文本和数字** → Times New Roman (TTF)

## 使用方法

### 1. 自动模式（推荐）

在表单填充时，系统会自动检测文本类型并选择字体：

```json
{
  "form_fields": [
    {
      "page_number": 1,
      "entry_text": {
        "text": "张三",  // 自动使用 STSong-Light (CIDFont)
        "font_size": 12
      }
    },
    {
      "page_number": 1,
      "entry_text": {
        "text": "John Smith",  // 自动使用 Times New Roman
        "font_size": 12
      }
    },
    {
      "page_number": 1,
      "entry_text": {
        "text": "ID: 12345",  // 自动使用 Times New Roman
        "font_size": 12
      }
    }
  ]
}
```

### 2. 手动注册字体

```python
from register_fonts import register_chinese_fonts, get_chinese_font_name, get_english_font_name

# 注册所有字体（优先 CIDFont）
registered = register_chinese_fonts()

# 获取推荐字体
chinese_font = get_chinese_font_name()  # STSong-Light (CIDFont)
english_font = get_english_font_name()  # Times New Roman
```

### 3. 在 reportlab 中使用 CIDFont（推荐）

```python
from reportlab.pdfgen import canvas
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.cidfonts import UnicodeCIDFont

# 注册 CIDFont（无需字体文件！）
pdfmetrics.registerFont(UnicodeCIDFont('STSong-Light'))

# 创建PDF
c = canvas.Canvas("output.pdf", pagesize=letter)

# 使用宋体绘制中文
c.setFont('STSong-Light', 14)
c.drawString(100, 750, "中文内容")

c.save()
```

### 4. 在 reportlab 中使用 TTF（备选）

```python
from reportlab.pdfgen import canvas
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont

# 注册 TTF 字体（需要字体文件）
pdfmetrics.registerFont(TTFont('NotoSerifCJK', '/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc'))
pdfmetrics.registerFont(TTFont('TimesNewRoman', '/usr/share/fonts/truetype/liberation/LiberationSerif-Regular.ttf'))

# 创建PDF
c = canvas.Canvas("output.pdf", pagesize=letter)

# 使用宋体绘制中文
c.setFont('NotoSerifCJK', 14)
c.drawString(100, 750, "中文内容")

# 使用新罗马绘制英文
c.setFont('TimesNewRoman', 14)
c.drawString(100, 720, "English Content 123")

c.save()
```

## 测试验证

运行测试脚本验证字体配置：

```bash
cd configs/microservice/bff-service/configs/agent-skills/anthropics/pdf
python scripts/test_chinese_fonts.py
```

测试内容包括：
1. 字体注册测试（CIDFont + TTF）
2. 中文检测测试
3. 字体选择测试
4. 文本渲染测试

## Docker 配置

字体已在 Dockerfile.wga-sandbox 中预装：

```dockerfile
RUN set -eux; \
    apt-get install -y --no-install-recommends \
        fonts-noto-cjk \
        fonts-wqy-zenhei \
        fonts-wqy-microhei \
        fonts-liberation \
        fontconfig; \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*;
```

**注意**：CIDFont (STSong-Light) 不需要安装字体文件，由 PDF 阅读器内置支持。TTF 字体作为备选方案。

## 常见问题

### Q: 为什么优先使用 CIDFont？

A: CIDFont 有以下优势：
1. **无需字体文件** - PDF 阅读器内置支持
2. **PDF 文件更小** - 不需要嵌入字体
3. **完美兼容性** - Adobe 标准 CID 字体
4. **处理速度更快** - 无需加载字体文件

### Q: 如何查看系统中可用的字体？

A: 运行以下命令：
```bash
python scripts/register_fonts.py
```

### Q: 如何确认字体是否正确注册？

A: 查看测试脚本的输出，或检查 PDF 渲染结果。

### Q: 混合文本如何处理？

A: 系统会检测文本中是否包含中文字符。如果包含中文，使用 STSong-Light (CIDFont)；否则使用 Times New Roman。

### Q: 可以手动指定字体吗？

A: 可以。在 fields.json 中指定 font 字段：
```json
{
  "entry_text": {
    "text": "内容",
    "font": "STSong-Light",  // 手动指定 CIDFont
    "font_size": 12
  }
}
```

### Q: CIDFont 支持哪些字体？

A: 常见的 CIDFont 包括：
- `STSong-Light` - 宋体
- `STHeiti` - 黑体
- `STKaiti` - 楷体
- `STFangsong` - 仿宋

## 技术细节

### 字体检测逻辑

```python
def contains_chinese(text):
    """检测文本中是否包含中文字符"""
    chinese_pattern = re.compile(r'[\u4e00-\u9fff]+')
    return bool(chinese_pattern.search(text))

def select_font_for_text(text):
    """根据文本内容选择字体"""
    if contains_chinese(text):
        return get_chinese_font(), "Chinese"  # STSong-Light (CIDFont)
    else:
        return get_english_font(), "English"  # Times New Roman
```

### 字体注册优先级

```python
def register_chinese_fonts():
    # 1. 优先注册 CIDFont（无需字体文件）
    pdfmetrics.registerFont(UnicodeCIDFont('STSong-Light'))
    
    # 2. 备选：注册 TTF 字体（需要字体文件）
    pdfmetrics.registerFont(TTFont('NotoSerifCJK', '/path/to/font.ttf'))
```

### CIDFont 字体说明

- **STSong-Light** 是 Adobe 提供的标准中文字体
- 属于 CIDFont 格式，专为 CJK（中日韩）字符设计
- 内置于所有符合标准的 PDF 阅读器
- 支持完整的 Unicode 中文字符集

## 相关文件

- [register_fonts.py](scripts/register_fonts.py) - 字体注册脚本
- [fill_pdf_form_with_annotations.py](scripts/fill_pdf_form_with_annotations.py) - 表单填充脚本
- [test_chinese_fonts.py](scripts/test_chinese_fonts.py) - 测试脚本
- [SKILL.md](SKILL.md) - 完整使用文档
- [forms.md](forms.md) - 表单填充指南
