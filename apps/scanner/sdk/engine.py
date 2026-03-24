"""
Shared Analyzer Engine (Singleton Pattern)
===========================================
Provides a single instance of Presidio AnalyzerEngine with custom recognizers.
This prevents RAM explosion by loading NLP models only once.

CRITICAL: Uses en_core_web_sm (Small) model to keep RAM under 1GB.
"""

import os
import yaml
from typing import Optional
from presidio_analyzer import AnalyzerEngine, RecognizerRegistry
from presidio_analyzer.nlp_engine import NlpEngineProvider


class SharedAnalyzerEngine:
    """
    Singleton wrapper for Presidio AnalyzerEngine.
    Ensures only one instance exists across all scanning threads.
    """
    
    _instance: Optional[AnalyzerEngine] = None
    _config = None
    
    @classmethod
    def get_engine(cls, config_path: str = None) -> AnalyzerEngine:
        """
        Get or create the singleton AnalyzerEngine instance.
        
        Args:
            config_path: Path to SDK configuration YAML
            
        Returns:
            Configured AnalyzerEngine instance
        """
        if cls._instance is None:
            cls._instance = cls._initialize_engine(config_path)
        return cls._instance
    
    @classmethod
    def _initialize_engine(cls, config_path: Optional[str]) -> AnalyzerEngine:
        """
        Initialize the AnalyzerEngine with configuration.
        
        Args:
            config_path: Path to config.yml
            
        Returns:
            Configured AnalyzerEngine
        """
        # Load configuration
        if config_path and os.path.exists(config_path):
            with open(config_path, 'r') as f:
                cls._config = yaml.safe_load(f)
        else:
            # Default configuration
            cls._config = {
                'model': {
                    'name': 'en_core_web_sm',
                    'lang_code': 'en'
                },
                'allow_list': []
            }
        
        # Get model configuration
        model_name = cls._config.get('model', {}).get('name', 'en_core_web_sm')
        lang_code = cls._config.get('model', {}).get('lang_code', 'en')
        
        # CRITICAL: Enforce small model
        if 'lg' in model_name.lower():
            raise ValueError(
                f"Large models are forbidden! Got: {model_name}. "
                f"Use en_core_web_sm to prevent RAM explosion."
            )
        
        print(f"[SDK] Initializing Presidio with model: {model_name}")
        
        # Configure NLP engine with small model (Presidio-required format)
        nlp_configuration = {
            "nlp_engine_name": "spacy",
            "models": [
                {
                    "lang_code": lang_code,
                    "model_name": model_name,
                }
            ]
        }
        
        nlp_engine = NlpEngineProvider(nlp_configuration=nlp_configuration).create_engine()
        
        # Create empty registry (we'll add custom recognizers later)
        # NOTE: Not loading predefined recognizers to avoid version compatibility issues
        registry = RecognizerRegistry()
        # registry.load_predefined_recognizers(nlp_engine=nlp_engine, languages=[lang_code])  # Disabled
        
        # Create analyzer engine
        analyzer = AnalyzerEngine(
            nlp_engine=nlp_engine,
            registry=registry,
            supported_languages=[lang_code]
        )
        
        # Register all custom recognizers for locked PII types
        cls._register_custom_recognizers(analyzer)
        
        print(f"[SDK] AnalyzerEngine initialized successfully")
        print(f"[SDK] Memory footprint: ~500-800MB (Small model)")
        
        return analyzer
    
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
        results = engine.analyze(text=text, language='en')
        print(f"\n✓ Test analysis: Found {len(results)} entities")
        for result in results:
            print(f"  - {result.entity_type}: {text[result.start:result.end]}")
        
    except Exception as e:
        print(f"✗ Error: {e}")
