import subprocess
import sys
from pathlib import Path


def register_chinese_fonts():
    from reportlab.pdfbase import pdfmetrics
    from reportlab.pdfbase.cidfonts import UnicodeCIDFont
    from reportlab.pdfbase.ttfonts import TTFont
    from reportlab.lib.fonts import addMapping
    
    registered_fonts = {}
    
    # 优先使用 CIDFont（无需字体文件，PDF阅读器内置）
    try:
        pdfmetrics.registerFont(UnicodeCIDFont('STSong-Light'))
        registered_fonts['STSong-Light'] = {
            'path': 'CIDFont (Built-in)',
            'description': '宋体 (CIDFont)'
        }
        print(f"✓ Registered CIDFont: STSong-Light (宋体) - Built-in, no file needed")
    except Exception as e:
        print(f"✗ Failed to register STSong-Light CIDFont: {e}")
    
    # 备选：使用 TTF 字体（需要字体文件）
    # 注意：跳过 .ttc 文件，因为 reportlab 不支持 TTC 格式
    font_dirs = [
        '/usr/share/fonts/truetype/noto',
        '/usr/share/fonts/truetype/wqy',
        '/usr/share/fonts/opentype/noto',
        '/usr/share/fonts/truetype/liberation',
        '/usr/share/fonts',
    ]
    
    fonts_to_register = {
        'NotoSerifCJK': {
            'files': {
                'regular': ['NotoSerifCJKsc-Regular.otf'],  # 使用 OTF，跳过 TTC
                'bold': ['NotoSerifCJKsc-Bold.otf'],
            },
            'family': 'Noto Serif CJK SC',
            'description': '宋体 (OTF)'
        },
        'NotoSansCJK': {
            'files': {
                'regular': ['NotoSansCJKsc-Regular.otf'],  # 使用 OTF，跳过 TTC
                'bold': ['NotoSansCJKsc-Bold.otf'],
            },
            'family': 'Noto Sans CJK',
            'description': '黑体 (OTF)'
        },
        'WenQuanYiZenHei': {
            'files': {
                'regular': ['WenQuanYiZenHei.ttf'],  # 使用 TTF
            },
            'family': 'WenQuanYi Zen Hei',
            'description': '文泉驿正黑 (TTF)'
        },
        'WenQuanYiMicroHei': {
            'files': {
                'regular': ['WenQuanYiMicroHei.ttf'],  # 使用 TTF
            },
            'family': 'WenQuanYi Micro Hei',
            'description': '文泉驿微米黑 (TTF)'
        },
        'TimesNewRoman': {
            'files': {
                'regular': ['LiberationSerif-Regular.ttf', 'TimesNewRoman.ttf'],
                'bold': ['LiberationSerif-Bold.ttf', 'TimesNewRomanBold.ttf'],
                'italic': ['LiberationSerif-Italic.ttf', 'TimesNewRomanItalic.ttf'],
                'bolditalic': ['LiberationSerif-BoldItalic.ttf', 'TimesNewRomanBoldItalic.ttf'],
            },
            'family': 'Liberation Serif',
            'description': '新罗马 (TTF)'
        }
    }
    
    for font_name, font_info in fonts_to_register.items():
        for style, filenames in font_info['files'].items():
            for filename in filenames:
                # 跳过 TTC 文件，reportlab 不支持
                if filename.endswith('.ttc'):
                    continue
                    
                found = False
                for font_dir in font_dirs:
                    font_path = Path(font_dir) / filename
                    if font_path.exists():
                        try:
                            pdfmetrics.registerFont(TTFont(font_name, str(font_path)))
                            registered_fonts[font_name] = {
                                'path': str(font_path),
                                'description': font_info.get('description', font_name)
                            }
                            print(f"✓ Registered: {font_name} ({font_info.get('description', '')}) from {font_path}")
                            found = True
                            break
                        except Exception as e:
                            print(f"✗ Failed to register {font_name} from {font_path}: {e}")
                if found:
                    break
    
    return registered_fonts


def list_available_fonts():
    result = subprocess.run(
        ['fc-list', ':lang=zh', 'family', 'file'],
        capture_output=True,
        text=True
    )
    
    if result.returncode == 0:
        print("\n📋 Available Chinese fonts in system:")
        print("=" * 60)
        for line in result.stdout.strip().split('\n'):
            if line:
                parts = line.split(':')
                if len(parts) >= 2:
                    font_name = parts[0].strip()
                    font_file = parts[1].strip()
                    print(f"  {font_name}")
                    print(f"    File: {font_file}")
        print("=" * 60)
    else:
        print("Warning: Could not list fonts with fc-list")


def get_chinese_font_name():
    preferred_fonts = [
        'STSong-Light',  # CIDFont (优先，无需字体文件)
        'NotoSerifCJK',
        'NotoSansCJK',
        'WenQuanYiZenHei',
        'WenQuanYiMicroHei',
    ]
    
    from reportlab.pdfbase import pdfmetrics
    
    for font_name in preferred_fonts:
        try:
            pdfmetrics.getFont(font_name)
            return font_name
        except:
            continue
    
    return 'Helvetica'


def get_english_font_name():
    """
    获取英文字体名称
    
    优先级：
    1. Times-Roman (PDF内置，无需注册)
    2. Liberation Serif (需要字体文件)
    3. Helvetica (PDF内置，无需注册)
    """
    # PDF 内置字体，无需注册
    # Times-Roman 是 PDF 标准字体，类似 Times New Roman
    built_in_fonts = [
        'Times-Roman',  # PDF 内置，类似新罗马
    ]
    
    from reportlab.pdfbase import pdfmetrics
    
    # 首先尝试内置字体
    for font_name in built_in_fonts:
        try:
            pdfmetrics.getFont(font_name)
            return font_name
        except:
            continue
    
    # 然后尝试注册的字体
    registered_fonts = ['TimesNewRoman']
    for font_name in registered_fonts:
        try:
            pdfmetrics.getFont(font_name)
            return font_name
        except:
            continue
    
    # 最后使用 Helvetica (PDF 内置)
    return 'Helvetica'


if __name__ == "__main__":
    print("🔧 Registering Chinese fonts for PDF generation...")
    print("=" * 60)
    
    registered = register_chinese_fonts()
    
    print(f"\n✓ Successfully registered {len(registered)} fonts:")
    for name, info in registered.items():
        print(f"  - {name} ({info['description']}): {info['path']}")
    
    list_available_fonts()
    
    chinese_font = get_chinese_font_name()
    english_font = get_english_font_name()
    
    print(f"\n💡 Recommended fonts:")
    print(f"  - Chinese (宋体): {chinese_font}")
    print(f"  - English (新罗马): {english_font}")
