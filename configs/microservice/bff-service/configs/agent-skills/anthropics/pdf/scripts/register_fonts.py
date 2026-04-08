import subprocess
import sys
from pathlib import Path


def register_chinese_fonts():
    from reportlab.pdfbase import pdfmetrics
    from reportlab.pdfbase.ttfonts import TTFont
    from reportlab.lib.fonts import addMapping
    
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
                'regular': ['NotoSerifCJK-Regular.ttc', 'NotoSerifCJKsc-Regular.otf'],
                'bold': ['NotoSerifCJK-Bold.ttc', 'NotoSerifCJKsc-Bold.otf'],
            },
            'family': 'Noto Serif CJK SC',
            'description': '宋体'
        },
        'NotoSansCJK': {
            'files': {
                'regular': ['NotoSansCJK-Regular.ttc', 'NotoSansCJKsc-Regular.otf'],
                'bold': ['NotoSansCJK-Bold.ttc', 'NotoSansCJKsc-Bold.otf'],
            },
            'family': 'Noto Sans CJK',
            'description': '黑体'
        },
        'WenQuanYiZenHei': {
            'files': {
                'regular': ['wqy-zenhei.ttc', 'WenQuanYiZenHei.ttf'],
            },
            'family': 'WenQuanYi Zen Hei',
            'description': '文泉驿正黑'
        },
        'WenQuanYiMicroHei': {
            'files': {
                'regular': ['wqy-microhei.ttc', 'WenQuanYiMicroHei.ttf'],
            },
            'family': 'WenQuanYi Micro Hei',
            'description': '文泉驿微米黑'
        },
        'TimesNewRoman': {
            'files': {
                'regular': ['LiberationSerif-Regular.ttf', 'TimesNewRoman.ttf'],
                'bold': ['LiberationSerif-Bold.ttf', 'TimesNewRomanBold.ttf'],
                'italic': ['LiberationSerif-Italic.ttf', 'TimesNewRomanItalic.ttf'],
                'bolditalic': ['LiberationSerif-BoldItalic.ttf', 'TimesNewRomanBoldItalic.ttf'],
            },
            'family': 'Liberation Serif',
            'description': '新罗马'
        }
    }
    
    registered_fonts = {}
    
    for font_name, font_info in fonts_to_register.items():
        for style, filenames in font_info['files'].items():
            for filename in filenames:
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
                            print(f"✓ Registered font: {font_name} ({font_info.get('description', '')}) from {font_path}")
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
    preferred_fonts = [
        'TimesNewRoman',
    ]
    
    from reportlab.pdfbase import pdfmetrics
    
    for font_name in preferred_fonts:
        try:
            pdfmetrics.getFont(font_name)
            return font_name
        except:
            continue
    
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
