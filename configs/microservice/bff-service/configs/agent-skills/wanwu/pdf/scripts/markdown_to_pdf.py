#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
改进的 PDF 生成工具
支持 markdown 格式、自动换行、正确的样式处理
"""

import sys
import re
from pathlib import Path

sys.path.append(str(Path(__file__).parent))

from reportlab.lib.pagesizes import A4
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, PageBreak
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import cm
from reportlab.lib.enums import TA_LEFT, TA_CENTER, TA_JUSTIFY
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.cidfonts import UnicodeCIDFont


def setup_fonts():
    """设置字体"""
    pdfmetrics.registerFont(UnicodeCIDFont('STSong-Light'))
    return 'STSong-Light'


def create_styles(font_name):
    """创建样式"""
    styles = getSampleStyleSheet()
    
    # 标题样式
    title_style = ParagraphStyle(
        'CustomTitle',
        parent=styles['Title'],
        fontName=font_name,
        fontSize=24,
        spaceAfter=30,
        alignment=TA_CENTER,
        leading=30
    )
    
    # 一级标题
    heading1_style = ParagraphStyle(
        'CustomHeading1',
        parent=styles['Heading1'],
        fontName=font_name,
        fontSize=18,
        spaceAfter=12,
        spaceBefore=20,
        leading=22,
        textColor='#1a1a1a'
    )
    
    # 二级标题
    heading2_style = ParagraphStyle(
        'CustomHeading2',
        parent=styles['Heading2'],
        fontName=font_name,
        fontSize=16,
        spaceAfter=10,
        spaceBefore=15,
        leading=20,
        textColor='#333333'
    )
    
    # 正文样式
    body_style = ParagraphStyle(
        'CustomBody',
        parent=styles['Normal'],
        fontName=font_name,
        fontSize=11,
        spaceAfter=10,
        leading=16,
        alignment=TA_JUSTIFY,
        firstLineIndent=22  # 首行缩进
    )
    
    # 列表项样式
    list_style = ParagraphStyle(
        'CustomList',
        parent=styles['Normal'],
        fontName=font_name,
        fontSize=11,
        spaceAfter=8,
        leading=16,
        leftIndent=20,
        bulletIndent=10
    )
    
    return {
        'title': title_style,
        'heading1': heading1_style,
        'heading2': heading2_style,
        'body': body_style,
        'list': list_style
    }


def parse_markdown_line(line, styles):
    """
    解析 markdown 行并返回对应的 Paragraph
    
    Args:
        line: markdown 文本行
        styles: 样式字典
    
    Returns:
        Paragraph 对象或 None
    """
    line = line.strip()
    
    if not line:
        return None
    
    # 标题处理
    if line.startswith('# '):
        return Paragraph(line[2:], styles['title'])
    elif line.startswith('## '):
        return Paragraph(line[3:], styles['heading1'])
    elif line.startswith('### '):
        return Paragraph(line[4:], styles['heading2'])
    
    # 列表处理
    if line.startswith('- ') or line.startswith('* '):
        content = line[2:]
        # 处理粗体
        content = re.sub(r'\*\*(.+?)\*\*', r'<b>\1</b>', content)
        return Paragraph(f'• {content}', styles['list'])
    
    # 普通段落
    # 处理粗体
    content = re.sub(r'\*\*(.+?)\*\*', r'<b>\1</b>', line)
    
    return Paragraph(content, styles['body'])


def markdown_to_pdf(markdown_text, output_path, title=None):
    """
    将 markdown 文本转换为 PDF
    
    Args:
        markdown_text: markdown 文本
        output_path: 输出 PDF 路径
        title: 文档标题（可选）
    """
    # 设置字体
    font_name = setup_fonts()
    
    # 创建样式
    styles = create_styles(font_name)
    
    # 创建文档
    doc = SimpleDocTemplate(
        output_path,
        pagesize=A4,
        rightMargin=2*cm,
        leftMargin=2*cm,
        topMargin=2*cm,
        bottomMargin=2*cm
    )
    
    # 构建内容
    story = []
    
    # 添加标题（如果有）
    if title:
        story.append(Paragraph(title, styles['title']))
        story.append(Spacer(1, 0.5*cm))
    
    # 解析 markdown
    lines = markdown_text.split('\n')
    
    for line in lines:
        paragraph = parse_markdown_line(line, styles)
        if paragraph:
            story.append(paragraph)
    
    # 生成 PDF
    doc.build(story)
    
    print(f"✅ PDF 已生成: {output_path}")
    return output_path


def markdown_file_to_pdf(markdown_file, output_path=None):
    """
    将 markdown 文件转换为 PDF
    
    Args:
        markdown_file: markdown 文件路径
        output_path: 输出 PDF 路径（可选，默认为同名 .pdf）
    """
    markdown_path = Path(markdown_file)
    
    if not markdown_path.exists():
        raise FileNotFoundError(f"Markdown 文件不存在: {markdown_file}")
    
    # 读取 markdown 内容
    with open(markdown_path, 'r', encoding='utf-8') as f:
        markdown_text = f.read()
    
    # 设置输出路径
    if not output_path:
        output_path = markdown_path.with_suffix('.pdf')
    
    # 提取标题（第一个 # 标题）
    title = None
    for line in markdown_text.split('\n'):
        if line.startswith('# '):
            title = line[2:].strip()
            break
    
    return markdown_to_pdf(markdown_text, output_path, title)


if __name__ == "__main__":
    # 测试示例
    test_markdown = """
# 广东省介绍

## 地理位置

广东省位于中国南部，地处珠江三角洲，东邻福建，北接江西、湖南，西连广西，南临南海。全省陆地面积约17.97万平方公里，海域面积约41.93万平方公里。

## 主要城市

**广州市**：广东省省会，华南地区政治、经济、文化中心。

**深圳市**：中国第一个经济特区，已发展成为科技创新中心。

## 经济发展

广东是中国第一经济大省，2023年地区生产总值超过12万亿元，连续多年位居全国首位。
"""
    
    output = "/tmp/test_improved_pdf.pdf"
    markdown_to_pdf(test_markdown, output, "广东省介绍")
