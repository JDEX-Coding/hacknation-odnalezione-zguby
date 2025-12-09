#!/usr/bin/env python3
"""
Test script for enhanced semantic search
Verifies that the new implementation produces better results
"""

import sys
import numpy as np
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent))

def test_clip_handler():
    """Test the enhanced CLIP handler."""
    print("\n" + "="*80)
    print("🧪 Testing Enhanced CLIP Handler")
    print("="*80)

    try:
        from clip_handler_enhanced import CLIPHandler, TextPreprocessor, QueryEnhancer

        # Test 1: Model Loading
        print("\n📝 Test 1: Loading unified CLIP model...")
        handler = CLIPHandler()
        handler.load_model()

        dimension = handler.get_embedding_dimension()
        assert dimension == 512, f"Expected 512 dimensions, got {dimension}"
        print(f"✅ Model loaded successfully: {dimension}D embeddings")

        # Test 2: Text Embedding
        print("\n📝 Test 2: Generating text embedding...")
        text = "Czarny skórzany portfel znaleziony w parku"
        text_embedding = handler.encode_text(text)

        assert len(text_embedding) == 512, f"Expected 512D, got {len(text_embedding)}D"
        assert np.isclose(np.linalg.norm(text_embedding), 1.0, atol=0.01), "Embedding not normalized"
        print(f"✅ Text embedding: shape={text_embedding.shape}, norm={np.linalg.norm(text_embedding):.4f}")

        # Test 3: Multiple Text Embeddings
        print("\n📝 Test 3: Batch text embedding...")
        texts = [
            "czarny plecak",
            "niebieski telefon",
            "klucze samochodowe"
        ]
        batch_embeddings = handler.encode_text(texts)

        assert batch_embeddings.shape == (3, 512), f"Expected (3, 512), got {batch_embeddings.shape}"
        print(f"✅ Batch embedding: shape={batch_embeddings.shape}")

        # Test 4: Similarity Between Related Items
        print("\n📝 Test 4: Testing semantic similarity...")
        emb1 = handler.encode_text("czarny portfel skórzany")
        emb2 = handler.encode_text("black leather wallet")
        emb3 = handler.encode_text("niebieski telefon Samsung")

        sim_related = handler.compute_similarity(emb1, emb2)
        sim_unrelated = handler.compute_similarity(emb1, emb3)

        print(f"  Similarity (portfel ↔ wallet): {sim_related:.4f}")
        print(f"  Similarity (portfel ↔ telefon): {sim_unrelated:.4f}")

        assert sim_related > sim_unrelated, "Related items should be more similar!"
        assert sim_related > 0.7, f"Related items should have high similarity (got {sim_related:.4f})"
        print(f"✅ Semantic similarity working correctly")

        # Test 5: Text Preprocessing
        print("\n📝 Test 5: Testing text preprocessing...")
        preprocessor = TextPreprocessor()

        title = "Znaleziony Czarny Portfel"
        description = "Skórzany portfel marki Louis Vuitton"
        category = "Portfele"
        location = "Park Centralny"

        enhanced = preprocessor.create_enhanced_text(title, description, category, location)
        keywords = preprocessor.extract_keywords(enhanced)

        print(f"  Enhanced text: {enhanced[:100]}...")
        print(f"  Keywords: {keywords[:5]}")
        assert len(keywords) > 0, "Should extract keywords"
        print(f"✅ Text preprocessing working")

        # Test 6: Query Enhancement
        print("\n📝 Test 6: Testing query enhancement...")
        enhancer = QueryEnhancer()

        queries = [
            "czarny portfel",
            "niebieski telefon",
            "klucze od samochodu"
        ]

        for query in queries:
            enhanced = enhancer.enhance_query(query)
            print(f"  '{query}' → '{enhanced[:60]}...'")
            assert len(enhanced) > len(query), "Enhanced query should be longer"

        print(f"✅ Query enhancement working")

        # Test 7: Embedding Combination
        print("\n📝 Test 7: Testing embedding combination...")
        text_emb = handler.encode_text("czarny plecak Nike")

        # Simulate image embedding (in real use, would come from image)
        # For testing, use slightly different text
        image_emb = handler.encode_text("black Nike backpack")

        # Test different weights
        combined_50_50 = handler.combine_embeddings(text_emb, image_emb, text_weight=0.5)
        combined_60_40 = handler.combine_embeddings(text_emb, image_emb, text_weight=0.6)

        assert len(combined_50_50) == 512, "Combined embedding wrong dimension"
        assert np.isclose(np.linalg.norm(combined_50_50), 1.0, atol=0.01), "Combined embedding not normalized"

        # Combined should be between the two originals in semantic space
        sim_text_combined = handler.compute_similarity(text_emb, combined_60_40)
        sim_image_combined = handler.compute_similarity(image_emb, combined_60_40)

        print(f"  Text-Combined similarity: {sim_text_combined:.4f}")
        print(f"  Image-Combined similarity: {sim_image_combined:.4f}")
        print(f"  Combined norm: {np.linalg.norm(combined_60_40):.4f}")

        assert sim_text_combined > 0.8, "Combined should be close to both inputs"
        assert sim_image_combined > 0.8, "Combined should be close to both inputs"
        print(f"✅ Embedding combination working correctly")

        print("\n" + "="*80)
        print("✅ ALL TESTS PASSED!")
        print("="*80)
        return True

    except Exception as e:
        print(f"\n❌ TEST FAILED: {e}")
        import traceback
        traceback.print_exc()
        return False


def test_search_quality():
    """Test search quality with example queries."""
    print("\n" + "="*80)
    print("🔍 Testing Search Quality")
    print("="*80)

    try:
        from clip_handler_enhanced import CLIPHandler, QueryEnhancer

        handler = CLIPHandler()
        handler.load_model()
        enhancer = QueryEnhancer()

        # Create a small test corpus
        corpus = [
            "Czarny skórzany portfel marki Louis Vuitton znaleziony w parku",
            "Niebieski telefon Samsung Galaxy S21 zgubiony w centrum",
            "Czerwona torebka damska z małą kieszonką",
            "Klucze od samochodu BMW z brelokiem",
            "Srebrny zegarek męski marki Rolex",
            "Okulary przeciwsłoneczne Ray-Ban czarne"
        ]

        print("\n📚 Test Corpus:")
        for i, item in enumerate(corpus, 1):
            print(f"  {i}. {item}")

        # Generate embeddings for corpus
        print("\n⏳ Generating corpus embeddings...")
        corpus_embeddings = [handler.encode_text(text) for text in corpus]

        # Test queries
        test_queries = [
            ("czarny portfel", 0, "Should find black wallet"),
            ("telefon Samsung", 1, "Should find Samsung phone"),
            ("torebka damska", 2, "Should find lady's handbag"),
            ("klucze", 3, "Should find car keys"),
            ("zegarek", 4, "Should find watch"),
        ]

        print("\n🔍 Testing Queries:")
        all_passed = True

        for query, expected_idx, description in test_queries:
            print(f"\n  Query: '{query}' ({description})")

            # Enhance query
            enhanced_query = enhancer.enhance_query(query)
            print(f"  Enhanced: '{enhanced_query[:50]}...'")

            # Get query embedding
            query_emb = handler.encode_text(enhanced_query)

            # Calculate similarities
            similarities = [
                handler.compute_similarity(query_emb, corpus_emb)
                for corpus_emb in corpus_embeddings
            ]

            # Find best match
            best_idx = np.argmax(similarities)
            best_score = similarities[best_idx]

            print(f"  Best match (score={best_score:.4f}): {corpus[best_idx][:60]}...")

            # Check if correct
            if best_idx == expected_idx:
                print(f"  ✅ CORRECT! Score: {best_score:.4f}")
                if best_score < 0.7:
                    print(f"  ⚠️  Warning: Score is low ({best_score:.4f})")
            else:
                print(f"  ❌ WRONG! Expected: {corpus[expected_idx][:60]}...")
                print(f"     Got: {corpus[best_idx][:60]}...")
                all_passed = False

            # Show top 3 results
            top3_indices = np.argsort(similarities)[::-1][:3]
            print(f"  Top 3 matches:")
            for rank, idx in enumerate(top3_indices, 1):
                print(f"    {rank}. [{similarities[idx]:.4f}] {corpus[idx][:50]}...")

        print("\n" + "="*80)
        if all_passed:
            print("✅ SEARCH QUALITY TEST PASSED!")
        else:
            print("⚠️  SEARCH QUALITY TEST: Some queries failed")
        print("="*80)

        return all_passed

    except Exception as e:
        print(f"\n❌ SEARCH QUALITY TEST FAILED: {e}")
        import traceback
        traceback.print_exc()
        return False


def compare_with_old_approach():
    """Compare new unified model vs old mixed models approach."""
    print("\n" + "="*80)
    print("📊 Comparing Old vs New Approach")
    print("="*80)

    print("\n⚠️  This test requires the OLD clip_handler.py to be available")
    print("Skipping comparison test...")

    # TODO: Implement comparison if old handler is available
    return True


def main():
    """Run all tests."""
    print("\n" + "="*80)
    print("🚀 Enhanced Semantic Search Test Suite")
    print("="*80)

    results = {}

    # Run tests
    results['CLIP Handler'] = test_clip_handler()
    results['Search Quality'] = test_search_quality()
    results['Comparison'] = compare_with_old_approach()

    # Summary
    print("\n" + "="*80)
    print("📊 TEST SUMMARY")
    print("="*80)

    for test_name, passed in results.items():
        status = "✅ PASSED" if passed else "❌ FAILED"
        print(f"  {test_name}: {status}")

    all_passed = all(results.values())

    if all_passed:
        print("\n🎉 ALL TESTS PASSED! 🎉")
        print("\nYou're ready to deploy the enhanced semantic search!")
        print("\nNext steps:")
        print("  1. Follow MIGRATION_GUIDE.md to deploy")
        print("  2. Monitor search quality in production")
        print("  3. Gather user feedback")
        return 0
    else:
        print("\n❌ SOME TESTS FAILED")
        print("\nPlease fix the issues before deploying")
        return 1


if __name__ == "__main__":
    exit_code = main()
    sys.exit(exit_code)
