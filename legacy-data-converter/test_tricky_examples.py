"""
Test legacy data converter with tricky example files.
Run this script to test the converter with complex, messy, real-world data.
"""

import json
from pathlib import Path
from text_extractor import TextExtractor
from nlp_converter import NLPConverter


def test_file(file_path: Path, description: str):
    """Test conversion of a single file."""
    print(f"\n{'='*80}")
    print(f"📄 Testing: {file_path.name}")
    print(f"📝 Description: {description}")
    print(f"{'='*80}\n")
    
    # Extract text
    extractor = TextExtractor()
    print("🔍 Step 1: Extracting text...")
    extracted = extractor.extract(str(file_path))
    
    print(f"✅ Extracted {len(extracted['text'])} characters of text")
    print(f"📊 Metadata: {extracted['metadata']}")
    
    # Show raw data structure if available
    if extracted.get('raw_data'):
        print(f"📦 Raw data items: {len(extracted['raw_data']) if isinstance(extracted['raw_data'], list) else 'N/A'}")
    
    # Convert to lost items
    converter = NLPConverter()
    print("\n🔄 Step 2: Converting to lost-items schema...")
    
    # Handle multiple items vs single item
    items = []
    raw_data = extracted.get('raw_data')
    
    if raw_data and isinstance(raw_data, list):
        print(f"Found {len(raw_data)} structured items")
        for i, row_data in enumerate(raw_data):
            row_extracted = {
                'text': extracted['text'].split('\n')[i] if i < len(extracted['text'].split('\n')) else '',
                'raw_data': [row_data],
                'metadata': extracted['metadata']
            }
            item = converter.convert(row_extracted, dataset_id=f"test-{file_path.stem}")
            items.append(item)
            print(f"  ✓ Item {i+1}: {item.get('title', 'N/A')[:50]}...")
    else:
        print("Converting single item from text")
        item = converter.convert(extracted, dataset_id=f"test-{file_path.stem}")
        items.append(item)
        print(f"  ✓ Title: {item.get('title', 'N/A')}")
        print(f"  ✓ Category: {item.get('category', 'N/A')}")
        print(f"  ✓ Location: {item.get('location', 'N/A')}")
    
    # Display results
    print(f"\n✅ Converted {len(items)} item(s)")
    print("\n📋 Results:")
    print("-" * 80)
    
    for i, item in enumerate(items, 1):
        print(f"\n🔹 Item {i}:")
        print(f"  ID: {item.get('id', 'N/A')}")
        print(f"  Title: {item.get('title', 'N/A')}")
        print(f"  Description: {item.get('description', 'N/A')[:100]}...")
        print(f"  Category: {item.get('category', 'N/A')}")
        print(f"  Location: {item.get('location', 'N/A')}")
        print(f"  Found Date: {item.get('found_date', 'N/A')}")
        print(f"  Contact Email: {item.get('contact_email', 'N/A')}")
        print(f"  Contact Phone: {item.get('contact_phone', 'N/A')}")
        print(f"  Status: {item.get('status', 'N/A')}")
    
    return items


def main():
    """Run tests on all tricky example files."""
    
    print("\n" + "="*80)
    print("🧪 LEGACY DATA CONVERTER - TRICKY EXAMPLES TEST")
    print("="*80)
    
    examples_dir = Path(__file__).parent / "examples"
    
    # Define test cases
    test_cases = [
        {
            "file": "messy_email.html",
            "description": "Email with HTML formatting, nested content, and lists"
        },
        {
            "file": "messy_report1.txt",
            "description": "Police report with informal structure and Polish date format"
        },
        {
            "file": "messy_report2.txt",
            "description": "Unstructured text with varying formats"
        },
        {
            "file": "messy_report3.txt",
            "description": "Mixed format report with inconsistent spacing"
        },
        {
            "file": "messy_legacy_system.json",
            "description": "Legacy JSON with non-standard field names and mixed formats"
        },
        {
            "file": "messy_legacy_export.xml",
            "description": "Legacy XML with nested structures and inconsistent naming"
        },
        {
            "file": "messy_spreadsheet.csv",
            "description": "CSV with merged cells, empty rows, and inconsistent formatting"
        },
    ]
    
    all_results = {}
    
    for test_case in test_cases:
        file_path = examples_dir / test_case["file"]
        
        if not file_path.exists():
            print(f"\n⚠️  Skipping {test_case['file']} - file not found")
            continue
        
        try:
            items = test_file(file_path, test_case["description"])
            all_results[test_case["file"]] = {
                "success": True,
                "items": items,
                "count": len(items)
            }
        except Exception as e:
            print(f"\n❌ Error processing {test_case['file']}: {e}")
            import traceback
            traceback.print_exc()
            all_results[test_case["file"]] = {
                "success": False,
                "error": str(e),
                "count": 0
            }
    
    # Summary
    print("\n" + "="*80)
    print("📊 TEST SUMMARY")
    print("="*80 + "\n")
    
    successful = sum(1 for r in all_results.values() if r["success"])
    total = len(all_results)
    total_items = sum(r.get("count", 0) for r in all_results.values())
    
    print(f"Files processed: {successful}/{total}")
    print(f"Total items converted: {total_items}")
    print(f"\nResults by file:")
    
    for file_name, result in all_results.items():
        status = "✅" if result["success"] else "❌"
        count = result.get("count", 0)
        print(f"  {status} {file_name}: {count} items")
        if not result["success"]:
            print(f"     Error: {result.get('error', 'Unknown error')}")
    
    # Save results to file
    output_file = Path(__file__).parent / "output" / "test_results.json"
    output_file.parent.mkdir(exist_ok=True)
    
    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(all_results, f, indent=2, ensure_ascii=False, default=str)
    
    print(f"\n💾 Full results saved to: {output_file}")
    print("\n" + "="*80)


if __name__ == "__main__":
    main()
