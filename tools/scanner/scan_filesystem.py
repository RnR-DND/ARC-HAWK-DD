#!/usr/bin/env python3
"""
scan_filesystem.py - Filesystem Scanner (Tool Layer)

Scans filesystem paths for PII using Scanner SDK.
Connection parameters: path, exclude_patterns[]
"""

import os
import sys
import json
import re
from pathlib import Path
from typing import Dict, Any, List, Optional
from dataclasses import dataclass


@dataclass
class FilesystemConfig:
    """Filesystem connection configuration"""
    path: str
    exclude_patterns: Optional[List[str]] = None
    max_file_size_mb: int = 100
    supported_extensions: Optional[List[str]] = None


class FilesystemScanner:
    """Scanner for filesystem data sources"""
    
    SUPPORTED_EXTENSIONS = {
        '.txt', '.csv', '.json', '.xml', '.yaml', '.yml',
        '.py', '.js', '.ts', '.go', '.java', '.c', '.cpp',
        '.md', '.rst', '.html', '.css',
        '.log', '.conf', '.cfg', '.ini', '.env',
        '.pdf', '.doc', '.docx'  # May require OCR
    }
    
    def __init__(self, config: FilesystemConfig):
        self.config = config
        self.exclude_patterns = config.exclude_patterns or []
        self._compile_exclude_regex()
    
    def _compile_exclude_regex(self):
        """Compile exclusion patterns to regex"""
        patterns = []
        for pattern in self.exclude_patterns:
            patterns.append(re.escape(pattern))
        self.exclude_regex = re.compile('|'.join(patterns)) if patterns else None
    
    def _should_exclude(self, path: str) -> bool:
        """Check if path should be excluded"""
        if self.exclude_regex and self.exclude_regex.search(path):
            return True
        return False
    
    def _is_supported(self, filepath: str) -> bool:
        """Check if file extension is supported"""
        ext = Path(filepath).suffix.lower()
        return ext in self.SUPPORTED_EXTENSIONS
    
    def _load_fingerprints(self) -> Dict[str, str]:
        """Load PII detection patterns"""
        fingerprint_path = "config/fingerprint.yml"
        if os.path.exists(fingerprint_path):
            # Load YAML fingerprint file
            import yaml
            with open(fingerprint_path, 'r') as f:
                data = yaml.safe_load(f)
                return data.get('patterns', {})
        return {}
    
    def scan_file(self, filepath: str) -> List[Dict[str, Any]]:
        """Scan a single file for PII"""
        findings = []
        
        try:
            with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
                content = f.read()
                
            fingerprints = self._load_fingerprints()
            
            for pattern_name, pattern in fingerprints.items():
                matches = re.finditer(pattern, content, re.IGNORECASE)
                
                for match in matches:
                    # Get context (50 chars before and after)
                    start = max(0, match.start() - 50)
                    end = min(len(content), match.end() + 50)
                    sample_text = content[start:end]
                    
                    finding = {
                        "host": "localhost",
                        "file_path": filepath,
                        "pattern_name": pattern_name,
                        "matches": [match.group()],
                        "sample_text": sample_text,
                        "profile": "fs_example",
                        "data_source": "fs",
                        "severity": self._get_severity(pattern_name),
                        "file_data": {
                            "file_size": os.path.getsize(filepath),
                            "extension": Path(filepath).suffix
                        }
                    }
                    findings.append(finding)
                    
        except Exception as e:
            print(f"Error scanning {filepath}: {e}", file=sys.stderr)
        
        return findings
    
    def _get_severity(self, pattern_name: str) -> str:
        """Determine severity based on pattern type"""
        high_severity = {"IN_AADHAAR", "IN_PAN", "CREDIT_CARD", "IN_PASSPORT"}
        medium_severity = {"EMAIL", "PHONE", "BANK_ACCOUNT", "IFSC"}
        
        if pattern_name in high_severity:
            return "Critical"
        elif pattern_name in medium_severity:
            return "High"
        return "Medium"
    
    def scan_directory(self, directory: Optional[str] = None) -> Dict[str, List[Dict[str, Any]]]:
        """Scan directory recursively for PII"""
        directory = directory or self.config.path
        all_findings = {"fs": []}
        
        for root, dirs, files in os.walk(directory):
            # Skip excluded directories
            dirs[:] = [d for d in dirs if not self._should_exclude(d)]
            
            for filename in files:
                filepath = os.path.join(root, filename)
                
                # Skip excluded files
                if self._should_exclude(filepath):
                    continue
                
                # Skip unsupported file types
                if not self._is_supported(filepath):
                    continue
                
                # Check file size
                try:
                    size_mb = os.path.getsize(filepath) / (1024 * 1024)
                    if size_mb > self.config.max_file_size_mb:
                        print(f"Skipping {filepath} (too large: {size_mb:.1f}MB)")
                        continue
                except OSError:
                    continue
                
                # Scan file
                findings = self.scan_file(filepath)
                all_findings["fs"].extend(findings)
        
        return all_findings


def execute(config: Dict[str, Any], pii_types: Optional[List[str]] = None, options: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Execute filesystem scan.
    
    Args:
        config: Connection configuration (path, exclude_patterns)
        pii_types: List of PII types to detect (optional)
        options: Scan options (optional)
    
    Returns:
        Dict with "fs" key containing findings list
    """
    fs_config = FilesystemConfig(
        path=config["path"],
        exclude_patterns=config.get("exclude_patterns", [])
    )
    
    scanner = FilesystemScanner(fs_config)
    results = scanner.scan_directory()
    
    # Filter by PII types if specified
    if pii_types:
        results["fs"] = [
            f for f in results["fs"]
            if f["pattern_name"] in pii_types
        ]
    
    return results


def main():
    """CLI entry point"""
    if len(sys.argv) < 2:
        print("Usage: scan_filesystem.py <connection_config.json>")
        sys.exit(1)
    
    with open(sys.argv[1], 'r') as f:
        config = json.load(f)
    
    results = execute(config)
    print(json.dumps(results, indent=2))


if __name__ == "__main__":
    main()
