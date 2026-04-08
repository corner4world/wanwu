import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))

from register_fonts import register_chinese_fonts, get_chinese_font_name, get_english_font_name, list_available_fonts

def test_font_registration():
    print("🧪 Testing Font Registration")
    print("=" * 60)
    
    registered = register_chinese_fonts()
    
    if registered:
        print(f"\n✅ SUCCESS: Registered {len(registered)} fonts")
        for name, info in registered.items():
            print(f"  - {name} ({info['description']}): {info['path']}")
    else:
        print("\n❌ FAILED: No fonts were registered")
        return False
    
    chinese_font = get_chinese_font_name()
    english_font = get_english_font_name()
    
    print(f"\n📝 Recommended fonts:")
    print(f"  - Chinese (宋体): {chinese_font}")
    print(f"  - English (新罗马): {english_font}")
    
    list_available_fonts()
    
    return True


def test_font_selection():
    print("\n🧪 Testing Font Selection")
    print("=" * 60)
    
    from fill_pdf_form_with_annotations import select_font_for_text
    
    test_cases = [
        ("张三", "Chinese", "宋体"),
        ("John Smith", "English", "新罗马"),
        ("123456", "English", "新罗马"),
        ("ID: 12345", "English", "新罗马"),
        ("你好World", "Chinese", "宋体"),
        ("测试Test测试", "Chinese", "宋体"),
    ]
    
    all_passed = True
    for text, expected_type, expected_font_desc in test_cases:
        font_name, font_type = select_font_for_text(text)
        status = "✅" if font_type == expected_type else "❌"
        print(f"{status} '{text}': {font_type} ({expected_font_desc})")
        if font_type != expected_type:
            all_passed = False
    
    return all_passed


def test_chinese_text_rendering():
    print("\n🧪 Testing Mixed Font Rendering")
    print("=" * 60)
    
    try:
        from reportlab.lib.pagesizes import letter
        from reportlab.pdfgen import canvas
        from reportlab.pdfbase import pdfmetrics
        from reportlab.pdfbase.ttfonts import TTFont
        
        chinese_font = get_chinese_font_name()
        english_font = get_english_font_name()
        
        if chinese_font == 'Helvetica' or english_font == 'Helvetica':
            print("⚠️  Required fonts not available")
            return False
        
        output_file = "/tmp/test_mixed_fonts.pdf"
        c = canvas.Canvas(output_file, pagesize=letter)
        
        c.setFont(chinese_font, 16)
        c.drawString(100, 750, "测试中文字体（宋体）")
        c.drawString(100, 720, "姓名：张三")
        
        c.setFont(english_font, 16)
        c.drawString(100, 690, "Test English Font (Times New Roman)")
        c.drawString(100, 660, "Name: John Smith")
        c.drawString(100, 630, "ID: 12345")
        
        c.save()
        
        print(f"✅ SUCCESS: Created test PDF with mixed fonts")
        print(f"📄 Output file: {output_file}")
        print(f"🔤 Chinese font (宋体): {chinese_font}")
        print(f"🔤 English font (新罗马): {english_font}")
        
        return True
        
    except Exception as e:
        print(f"❌ FAILED: {e}")
        import traceback
        traceback.print_exc()
        return False


def test_chinese_detection():
    print("\n🧪 Testing Chinese Character Detection")
    print("=" * 60)
    
    from fill_pdf_form_with_annotations import contains_chinese
    
    test_cases = [
        ("Hello World", False),
        ("你好世界", True),
        ("Hello 世界", True),
        ("123456", False),
        ("测试Test测试", True),
    ]
    
    all_passed = True
    for text, expected in test_cases:
        result = contains_chinese(text)
        status = "✅" if result == expected else "❌"
        print(f"{status} '{text}': {result} (expected: {expected})")
        if result != expected:
            all_passed = False
    
    return all_passed


if __name__ == "__main__":
    print("🚀 Running Font Support Tests")
    print("=" * 60)
    
    results = []
    
    results.append(("Font Registration", test_font_registration()))
    results.append(("Chinese Detection", test_chinese_detection()))
    results.append(("Font Selection", test_font_selection()))
    results.append(("Text Rendering", test_chinese_text_rendering()))
    
    print("\n" + "=" * 60)
    print("📊 Test Results Summary")
    print("=" * 60)
    
    for test_name, passed in results:
        status = "✅ PASS" if passed else "❌ FAIL"
        print(f"{status}: {test_name}")
    
    all_passed = all(passed for _, passed in results)
    
    print("=" * 60)
    if all_passed:
        print("🎉 All tests passed!")
        sys.exit(0)
    else:
        print("⚠️  Some tests failed")
        sys.exit(1)
