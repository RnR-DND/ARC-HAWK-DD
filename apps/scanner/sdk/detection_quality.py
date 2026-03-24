"""
Detection Quality Metrics - Track PII Detection Performance
===========================================================
Monitors detection quality to identify false positives and improve accuracy.

Metrics Tracked:
- Test data detection rate
- Confidence score distribution
- Validation pass/fail rates per PII type
- Context quality scores
"""

from typing import Dict, List, Optional
from dataclasses import dataclass, field
from datetime import datetime
from collections import defaultdict
import json


@dataclass
class DetectionMetrics:
    """Metrics for a single detection run"""
    timestamp: datetime
    total_detections: int = 0
    validated_findings: int = 0
    rejected_findings: int = 0
    test_data_filtered: int = 0
    context_rejected: int = 0
    
    # Per-PII-type metrics
    detections_by_type: Dict[str, int] = field(default_factory=lambda: defaultdict(int))
    validations_by_type: Dict[str, int] = field(default_factory=lambda: defaultdict(int))
    rejections_by_type: Dict[str, int] = field(default_factory=lambda: defaultdict(int))
    
    # Confidence distribution
    confidence_scores: List[float] = field(default_factory=list)
    
    # Context quality
    context_adjustments: List[float] = field(default_factory=list)  # Confidence changes
    
    def add_detection(self, pii_type: str):
        """Record a detection"""
        self.total_detections += 1
        self.detections_by_type[pii_type] += 1
    
    def add_validation(self, pii_type: str, confidence: float):
        """Record a successful validation"""
        self.validated_findings += 1
        self.validations_by_type[pii_type] += 1
        self.confidence_scores.append(confidence)
    
    def add_rejection(self, pii_type: str, reason: str):
        """Record a rejection"""
        self.rejected_findings += 1
        self.rejections_by_type[pii_type] += 1
        
        if "test data" in reason.lower():
            self.test_data_filtered += 1
        elif "context" in reason.lower():
            self.context_rejected += 1
    
    def add_context_adjustment(self, base_confidence: float, adjusted_confidence: float):
        """Record a confidence adjustment"""
        adjustment = adjusted_confidence - base_confidence
        self.context_adjustments.append(adjustment)
    
    def get_validation_rate(self) -> float:
        """Get overall validation rate"""
        if self.total_detections == 0:
            return 0.0
        return self.validated_findings / self.total_detections
    
    def get_rejection_rate(self) -> float:
        """Get overall rejection rate"""
        if self.total_detections == 0:
            return 0.0
        return self.rejected_findings / self.total_detections
    
    def get_test_data_rate(self) -> float:
        """Get test data detection rate"""
        if self.rejected_findings == 0:
            return 0.0
        return self.test_data_filtered / self.rejected_findings
    
    def get_average_confidence(self) -> float:
        """Get average confidence score"""
        if not self.confidence_scores:
            return 0.0
        return sum(self.confidence_scores) / len(self.confidence_scores)
    
    def get_average_context_adjustment(self) -> float:
        """Get average context adjustment"""
        if not self.context_adjustments:
            return 0.0
        return sum(self.context_adjustments) / len(self.context_adjustments)
    
    def to_dict(self) -> dict:
        """Convert to dictionary for serialization"""
        return {
            "timestamp": self.timestamp.isoformat(),
            "summary": {
                "total_detections": self.total_detections,
                "validated_findings": self.validated_findings,
                "rejected_findings": self.rejected_findings,
                "validation_rate": f"{self.get_validation_rate():.2%}",
                "rejection_rate": f"{self.get_rejection_rate():.2%}",
            },
            "rejection_breakdown": {
                "test_data_filtered": self.test_data_filtered,
                "context_rejected": self.context_rejected,
                "test_data_rate": f"{self.get_test_data_rate():.2%}",
            },
            "confidence": {
                "average": f"{self.get_average_confidence():.2f}",
                "min": f"{min(self.confidence_scores):.2f}" if self.confidence_scores else "N/A",
                "max": f"{max(self.confidence_scores):.2f}" if self.confidence_scores else "N/A",
            },
            "context_adjustments": {
                "average": f"{self.get_average_context_adjustment():.3f}",
                "total_adjustments": len(self.context_adjustments),
            },
            "by_pii_type": {
                pii_type: {
                    "detections": self.detections_by_type[pii_type],
                    "validated": self.validations_by_type[pii_type],
                    "rejected": self.rejections_by_type[pii_type],
                    "validation_rate": f"{self.validations_by_type[pii_type] / self.detections_by_type[pii_type]:.2%}"
                        if self.detections_by_type[pii_type] > 0 else "N/A"
                }
                for pii_type in self.detections_by_type.keys()
            }
        }


class DetectionQualityTracker:
    """
    Tracks detection quality metrics over time.
    """
    
    def __init__(self):
        self.current_metrics: Optional[DetectionMetrics] = None
        self.historical_metrics: List[DetectionMetrics] = []
    
    def start_new_run(self):
        """Start tracking a new detection run"""
        if self.current_metrics:
            self.historical_metrics.append(self.current_metrics)
        
        self.current_metrics = DetectionMetrics(timestamp=datetime.now())
    
    def record_detection(self, pii_type: str):
        """Record a detection"""
        if not self.current_metrics:
            self.start_new_run()
        self.current_metrics.add_detection(pii_type)
    
    def record_validation(self, pii_type: str, confidence: float):
        """Record a successful validation"""
        if not self.current_metrics:
            self.start_new_run()
        self.current_metrics.add_validation(pii_type, confidence)
    
    def record_rejection(self, pii_type: str, reason: str):
        """Record a rejection"""
        if not self.current_metrics:
            self.start_new_run()
        self.current_metrics.add_rejection(pii_type, reason)
    
    def record_context_adjustment(self, base_confidence: float, adjusted_confidence: float):
        """Record a confidence adjustment"""
        if not self.current_metrics:
            self.start_new_run()
        self.current_metrics.add_context_adjustment(base_confidence, adjusted_confidence)
    
    def get_current_metrics(self) -> Optional[DetectionMetrics]:
        """Get current run metrics"""
        return self.current_metrics
    
    def get_summary(self) -> dict:
        """Get summary of current run"""
        if not self.current_metrics:
            return {"error": "No active detection run"}
        
        return self.current_metrics.to_dict()
    
    def save_report(self, filepath: str):
        """Save metrics report to file"""
        if not self.current_metrics:
            return
        
        report = self.current_metrics.to_dict()
        
        with open(filepath, 'w') as f:
            json.dump(report, f, indent=2)
        
        print(f"📊 Detection quality report saved to {filepath}")
    
    def print_summary(self):
        """Print summary to console"""
        if not self.current_metrics:
            print("No active detection run")
            return
        
        m = self.current_metrics
        
        print("\n" + "="*60)
        print("DETECTION QUALITY SUMMARY")
        print("="*60)
        
        print(f"\n📊 Overall Statistics:")
        print(f"  Total Detections: {m.total_detections}")
        print(f"  Validated: {m.validated_findings} ({m.get_validation_rate():.1%})")
        print(f"  Rejected: {m.rejected_findings} ({m.get_rejection_rate():.1%})")
        
        print(f"\n🚫 Rejection Breakdown:")
        print(f"  Test Data Filtered: {m.test_data_filtered}")
        print(f"  Context Rejected: {m.context_rejected}")
        
        print(f"\n📈 Confidence Scores:")
        print(f"  Average: {m.get_average_confidence():.2f}")
        if m.confidence_scores:
            print(f"  Range: {min(m.confidence_scores):.2f} - {max(m.confidence_scores):.2f}")
        
        print(f"\n🎯 Context Adjustments:")
        print(f"  Average Adjustment: {m.get_average_context_adjustment():+.3f}")
        print(f"  Total Adjustments: {len(m.context_adjustments)}")
        
        print(f"\n📋 By PII Type:")
        for pii_type in sorted(m.detections_by_type.keys()):
            detections = m.detections_by_type[pii_type]
            validated = m.validations_by_type[pii_type]
            rejected = m.rejections_by_type[pii_type]
            rate = validated / detections if detections > 0 else 0
            print(f"  {pii_type:20s}: {validated:3d}/{detections:3d} validated ({rate:.1%})")
        
        print("="*60 + "\n")


# Global tracker instance
_quality_tracker = DetectionQualityTracker()


def get_quality_tracker() -> DetectionQualityTracker:
    """Get the global quality tracker instance"""
    return _quality_tracker


if __name__ == "__main__":
    print("=== Detection Quality Tracker Test ===\n")
    
    tracker = DetectionQualityTracker()
    tracker.start_new_run()
    
    # Simulate detections
    tracker.record_detection("IN_AADHAAR")
    tracker.record_validation("IN_AADHAAR", 0.95)
    tracker.record_context_adjustment(0.95, 0.92)
    
    tracker.record_detection("IN_PHONE")
    tracker.record_rejection("IN_PHONE", "Rejected: Test data pattern detected")
    
    tracker.record_detection("EMAIL_ADDRESS")
    tracker.record_validation("EMAIL_ADDRESS", 0.88)
    tracker.record_context_adjustment(0.88, 0.75)
    
    tracker.record_detection("CREDIT_CARD")
    tracker.record_rejection("CREDIT_CARD", "Rejected: Low confidence after context analysis")
    
    # Print summary
    tracker.print_summary()
    
    # Save report
    tracker.save_report("detection_quality_report.json")
