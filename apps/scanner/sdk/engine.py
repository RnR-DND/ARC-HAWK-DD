"""
Shared Analyzer Engine (Singleton Pattern)
===========================================
Provides a single instance of Presidio AnalyzerEngine with custom recognizers.
This prevents RAM explosion by loading NLP models only once.

CRITICAL: Uses en_core_web_sm (Small) model to keep RAM under 1GB.
"""

import os
import yaml
import threading
from typing import Optional
from presidio_analyzer import AnalyzerEngine, RecognizerRegistry
from presidio_analyzer.nlp_engine import NlpEngineProvider

class SharedAnalyzerEngine:
    _engine = None
    _lock = threading.Lock()

    @classmethod
    def get_engine(cls):
        if cls._engine is None:
            with cls._lock:
                if cls._engine is None:  # double-checked locking

                    print("[SDK] Initializing Presidio with model: en_core_web_sm")

                    nlp_configuration = {
                        "nlp_engine_name": "spacy",
                        "models": [
                            {
                                "lang_code": "en",
                                "model_name": "en_core_web_sm",
                            }
                        ]
                    }

                    nlp_engine = NlpEngineProvider(
                        nlp_configuration=nlp_configuration
                    ).create_engine()

                    registry = RecognizerRegistry()
                    registry._recognizers = []   # ✅ clean registry

                    engine = AnalyzerEngine(
                        nlp_engine=nlp_engine,
                        registry=registry,
                        supported_languages=["en"]
                    )

                    # 🚨 HARD reset after init
                    engine.registry._recognizers = []
                    engine.registry.recognizers = []

                    cls._register_custom_recognizers(engine)

                    cls._engine = engine

                    print("[SDK] Engine initialized ONCE")

        return cls._engine

    @classmethod
    def _register_custom_recognizers(cls, analyzer: AnalyzerEngine) -> None:
        """
        Register all custom recognizers for the 11 locked PII types.
        
        Args:
            analyzer: AnalyzerEngine instance to register recognizers with
        """
        print("[SDK] Registering custom recognizers for 11 locked PIIs...")
        
        # Import all custom recognizers
        from sdk.recognizers.aadhaar import AadhaarRecognizer
        from sdk.recognizers.pan import PANRecognizer
        from sdk.recognizers.credit_card import CreditCardRecognizer
        from sdk.recognizers.passport import IndianPassportRecognizer
        from sdk.recognizers.upi import UPIRecognizer
        from sdk.recognizers.ifsc import IFSCRecognizer
        from sdk.recognizers.bank_account import BankAccountRecognizer
        from sdk.recognizers.phone import IndianPhoneRecognizer
        from sdk.recognizers.email import EmailRecognizer
        from sdk.recognizers.voter_id import VoterIDRecognizer
        from sdk.recognizers.driving_license import DrivingLicenseRecognizer
        
        # Register each recognizer
        recognizers = [
            AadhaarRecognizer(),
            PANRecognizer(),
            CreditCardRecognizer(),
            IndianPassportRecognizer(),
            UPIRecognizer(),
            IFSCRecognizer(),
            BankAccountRecognizer(),
            IndianPhoneRecognizer(),
            EmailRecognizer(),
            VoterIDRecognizer(),
            DrivingLicenseRecognizer(),
        ]
        for recognizer in recognizers:
            analyzer.registry.add_recognizer(recognizer)
            print(f"  ✓ Registered: {recognizer.name}")
        entities = []
        for r in analyzer.registry.recognizers:
            if hasattr(r, "supported_entity"):
                entities.append(r.supported_entity)
            elif hasattr(r, "supported_entities"):
                entities.extend(r.supported_entities)
            else:
                entities.append(type(r).__name__)

        print("DEBUG ENTITIES:", entities)
        print(f"[SDK] Registered {len(recognizers)} custom recognizers")
    
    @classmethod
    def add_recognizer(cls, recognizer) -> None:
        """
        Add a custom recognizer to the engine.
        
        Args:
            recognizer: Custom PatternRecognizer instance
        """
        if cls._instance is None:
            raise RuntimeError("Engine not initialized. Call get_engine() first.")
        
        cls._instance.registry.add_recognizer(recognizer)
        print(f"[SDK] Added custom recognizer: {recognizer.name}")
    
    @classmethod
    def get_config(cls) -> dict:
        """Get the loaded configuration."""
        return cls._config or {}
    
    @classmethod
    def reset(cls) -> None:
        """Reset the singleton (for testing)."""
        cls._instance = None
        cls._config = None


if __name__ == "__main__":
    print("=== SharedAnalyzerEngine Test ===\n")
    
    # Test initialization
    try:
        engine = SharedAnalyzerEngine.get_engine()

        print(f"✓ Engine initialized")
        print(f"✓ Supported languages: {engine.supported_languages}")
        
        # Test singleton pattern
        engine2 = SharedAnalyzerEngine.get_engine()
        assert engine is engine2, "Not a singleton!"
        print(f"✓ Singleton pattern working")
        
        # Test analysis
        text = "My email is test@example.com"
        results = engine.analyze(text=text, language='en', entities=engine.get_supported_entities())
        print(f"\n✓ Test analysis: Found {len(results)} entities")
        for result in results:
            print(f"  - {result.entity_type}: {text[result.start:result.end]}")
        
    except Exception as e:
        print(f"✗ Error: {e}")
