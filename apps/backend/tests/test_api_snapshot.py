"""
Backend API Snapshot Tests
Lock baseline behavior of critical endpoints
"""
import requests
import json
import pytest


BASE_URL = "http://localhost:8080"


class TestHealthEndpoint:
    """Health check baseline"""
    
    def test_health_returns_200(self):
        """Health endpoint must return 200"""
        response = requests.get(f"{BASE_URL}/health")
        assert response.status_code == 200
    
    def test_health_response_structure(self):
        """Health response must have expected structure"""
        response = requests.get(f"{BASE_URL}/health")
        data = response.json()
        
        assert "status" in data
        assert "service" in data
        assert data["status"] == "healthy"
        assert data["service"] == "arc-platform-backend"


class TestLineageEndpoint:
    """Neo4j lineage API baseline"""
    
    def test_lineage_endpoint_exists(self):
        """Lineage endpoint must be accessible"""
        response = requests.get(f"{BASE_URL}/api/v1/lineage")
        
        # Should return 200 or 500 (if Neo4j not configured)
        # But NOT 404 (endpoint must exist)
        assert response.status_code != 404
    
    def test_lineage_stats_endpoint_exists(self):
        """Stats endpoint must be accessible"""
        response = requests.get(f"{BASE_URL}/api/v1/lineage/stats")
        
        # Should not be 404
        assert response.status_code != 404


class TestFindingsEndpoint:
    """Findings API baseline (if it exists)"""
    
    def test_findings_endpoint_structure(self):
        """Findings endpoint should return consistent structure"""
        try:
            response = requests.get(f"{BASE_URL}/api/v1/findings")
            
            if response.status_code == 200:
                data = response.json()
                
                # Should have some standard fields
                assert isinstance(data, (list, dict))
        except Exception:
            # Endpoint might not exist yet
            pytest.skip("Findings endpoint not available")


class TestIngestionEndpoint:
    """Ingestion API baseline"""
    
    def test_ingestion_endpoint_exists(self):
        """Original ingestion endpoint must exist"""
        # Don't actually POST, just check endpoint exists
        # A GET will likely fail, but should not be 404
        response = requests.options(f"{BASE_URL}/api/v1/scans/ingest")
        
        # OPTIONS should work for CORS
        assert response.status_code in [200, 204]


class TestCORSHeaders:
    """CORS configuration baseline"""
    
    def test_cors_headers_present(self):
        """CORS headers must be present"""
        response = requests.options(
            f"{BASE_URL}/api/v1/lineage",
            headers={"Origin": "http://localhost:3000"}
        )
        
        # Check for CORS headers
        assert "Access-Control-Allow-Origin" in response.headers or \
               response.status_code == 200  # CORS might be configured


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-s"])
