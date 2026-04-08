# PDF 字体配置说明

## 概述

本 PDF skill 已配置智能字体选择功能，支持中英文混合文本的自动字体选择。

## 字体配置

### 已安装字体

系统已预装以下字体：

| 字体名称 | 类型 | 用途 | 说明 |
|---------|------|------|------|
| Noto Serif CJK SC | 宋体 | 中文文本 | 推荐用于中文内容 |
| Liberation Serif | 新罗马 | 英文/数字 | Times New Roman 的开源替代 |
| Noto Sans CJK | 黑体 | 中文文本 | 备选中文字体 |
| WenQuanYi Zen Hei | 文泉驿正黑 | 中文文本 | 备选中文字体 |
| WenQuanYi Micro Hei | 文泉驿微米黑 | 中文文本 | 轻量级中文字体 |

### 字体选择规则

系统自动根据文本内容选择合适的字体：

- **中文文本** → 宋体
- **英文文本和数字** → 新罗马

## 使用方法

### 1. 自动模式（推荐）

在表单填充时，系统会自动检测文本类型并选择字体：

```json
{
  "form_fields": [
    {
      "page_number": 1,
      "entry_text": {
        "text": "张三",  // 自动使用宋体
        "font_size": 12
      }
    },
    {
      "page_number": 1,
      "entry_text": {
        "text": "John Smith",  // 自动使用新罗马
        "font_size": 12
      }
    },
    {
      "page_number": 1,
      "entry_text": {
        "text": "ID: 12345",  // 自动使用新罗马
        "font_size": 12
      }
    }
  ]
}
```

### 2. 手动注册字体

```python
from register_fonts import register_chinese_fonts, get_chinese_font_name, get_english_font_name

# 注册所有字体
registered = register_chinese_fonts()

# 获取推荐字体
chinese_font = get_chinese_font_name()  # 宋体
english_font = get_english_font_name()  # 新罗马
```

### 3. 在 reportlab 中使用

```python
from reportlab.pdfgen import canvas
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont

# 注册字体
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
1. 字体注册测试
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

## 常见问题

### Q: 如何查看系统中可用的字体？

A: 运行以下命令：
```bash
python scripts/register_fonts.py
```

### Q: 如何确认字体是否正确注册？

A: 查看测试脚本的输出，或检查 PDF 渲染结果。

### Q: 混合文本如何处理？

A: 系统会检测文本中是否包含中文字符。如果包含中文，使用宋体；否则使用新罗马。

### Q: 可以手动指定字体吗？

A: 可以。在 fields.json 中指定 font 字段：
```json
{
  "entry_text": {
    "text": "内容",
    "font": "NotoSerifCJK",  // 手动指定字体
    "font_size": 12
  }
}
```

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
        return get_chinese_font(), "Chinese"  # 宋体
    else:
        return get_english_font(), "English"  # 新罗马
```

### 字体文件位置

- 宋体: `/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc`
- 新罗马: `/usr/share/fonts/truetype/liberation/LiberationSerif-Regular.ttf`

## 相关文件

- [register_fonts.py](scripts/register_fonts.py) - 字体注册脚本
- [fill_pdf_form_with_annotations.py](scripts/fill_pdf_form_with_annotations.py) - 表单填充脚本
- [test_chinese_fonts.py](scripts/test_chinese_fonts.py) - 测试脚本
- [SKILL.md](SKILL.md) - 完整使用文档
- [forms.md](forms.md) - 表单填充指南
