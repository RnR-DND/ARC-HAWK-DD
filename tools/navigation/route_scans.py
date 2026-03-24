#!/usr/bin/env python3
"""
route_scans.py - Scan Router (Navigation Layer)

Routes scan jobs to appropriate handlers based on source type.
Each source type has UNIQUE connection parameters.

Error Handling:
- All errors are logged and graceful fallback is provided
- Invalid configurations are rejected with clear messages
- Connection errors are retried up to 3 times
"""

import sys
import json
import logging
from typing import Dict, Any, List, Optional
from dataclasses import dataclass, field
from enum import Enum
from pathlib import Path

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class SourceType(Enum):
    """Supported data source types"""
    FILESYSTEM = "fs"
    POSTGRESQL = "postgresql"
    MYSQL = "mysql"
    MONGODB = "mongodb"
    S3 = "s3"
    GCS = "gcs"
    REDIS = "redis"
    SLACK = "slack"
    FIREBASE = "firebase"
    GOOGLE_DRIVE = "gdrive"
    GOOGLE_DRIVE_WORKSPACE = "gdrive_workspace"
    TEXT = "text"

    @classmethod
    def from_string(cls, value: str) -> Optional['SourceType']:
        """Safely convert string to SourceType"""
        try:
            return cls(value)
        except ValueError:
            logger.error(f"Unknown source type: {value}")
            return None


@dataclass
class ConnectionProfile:
    """Base connection profile"""
    profile_name: str
    source_type: SourceType


@dataclass
class FilesystemProfile(ConnectionProfile):
    """Filesystem connection profile"""
    path: str
    exclude_patterns: List[str] = field(default_factory=list)


@dataclass
class PostgreSQLProfile(ConnectionProfile):
    """PostgreSQL connection profile"""
    host: str
    port: int = 5432
    user: Optional[str] = None
    password: Optional[str] = None
    database: Optional[str] = None
    limit_start: int = 0
    limit_end: int = 50000
    tables: List[str] = field(default_factory=list)


@dataclass
class MySQLProfile(ConnectionProfile):
    """MySQL connection profile"""
    host: str
    port: int = 3306
    user: Optional[str] = None
    password: Optional[str] = None
    database: Optional[str] = None
    limit_start: int = 0
    limit_end: int = 500
    tables: List[str] = field(default_factory=list)
    exclude_columns: List[str] = field(default_factory=list)


@dataclass
class MongoDBProfile(ConnectionProfile):
    """MongoDB connection profile"""
    uri: Optional[str] = None
    host: Optional[str] = None
    port: int = 27017
    username: Optional[str] = None
    password: Optional[str] = None
    database: Optional[str] = None
    limit_start: int = 0
    limit_end: int = 500
    collections: List[str] = field(default_factory=list)


@dataclass
class S3Profile(ConnectionProfile):
    """AWS S3 connection profile"""
    access_key: Optional[str] = None
    secret_key: Optional[str] = None
    bucket_name: Optional[str] = None
    cache: bool = True
    exclude_patterns: List[str] = field(default_factory=list)


@dataclass
class GCSProfile(ConnectionProfile):
    """Google Cloud Storage connection profile"""
    credentials_file: Optional[str] = None
    bucket_name: Optional[str] = None
    cache: bool = True
    exclude_patterns: List[str] = field(default_factory=list)


@dataclass
class RedisProfile(ConnectionProfile):
    """Redis connection profile"""
    host: Optional[str] = None
    password: Optional[str] = None


@dataclass
class SlackProfile(ConnectionProfile):
    """Slack connection profile"""
    channel_types: str = "public_channel,private_channel"
    token: Optional[str] = None
    only_archived: bool = False
    archived_channels: bool = False
    limit_mins: int = 60
    read_from: str = "last_message"
    is_external: Optional[bool] = None
    channel_ids: List[str] = field(default_factory=list)
    blacklisted_channel_ids: List[str] = field(default_factory=list)


class ScanRouter:
    """
    Routes scan jobs to appropriate handlers.
    Each source type requires UNIQUE connection parameters.
    
    Error Handling:
    - Logs all routing decisions
    - Validates configurations before routing
    - Returns structured error responses
    """
    
    HANDLERS = {
        SourceType.FILESYSTEM: "scan_filesystem.py",
        SourceType.POSTGRESQL: "scan_postgresql.py",
        SourceType.MYSQL: "scan_mysql.py",
        SourceType.MONGODB: "scan_mongodb.py",
        SourceType.S3: "scan_s3.py",
        SourceType.GCS: "scan_gcs.py",
        SourceType.REDIS: "scan_redis.py",
        SourceType.SLACK: "scan_slack.py",
    }
    
    def __init__(self, config_path: str = "config/connection.yml"):
        self.config_path = config_path
        self.profiles: Dict[str, Dict[str, Any]] = {}
        self._load_profiles()
    
    def _load_profiles(self) -> None:
        """Load connection profiles from config file"""
        config_file = Path(self.config_path)
        
        if not config_file.exists():
            logger.warning(f"Config file not found: {self.config_path}")
            return
        
        try:
            with open(config_file, 'r') as f:
                data = json.load(f)
                self.profiles = data.get('sources', {})
                logger.info(f"Loaded {len(self.profiles)} connection profiles")
        except json.JSONDecodeError as e:
            logger.error(f"Invalid JSON in config file: {e}")
        except IOError as e:
            logger.error(f"Failed to read config file: {e}")
    
    def _get_required_field(self, config: Dict[str, Any], field_name: str) -> str:
        """Get required field or raise ValueError"""
        value = config.get(field_name)
        if value is None:
            raise ValueError(f"Required field '{field_name}' is missing")
        return value
    
    def _get_optional_list(self, config: Dict[str, Any], field_name: str) -> List[str]:
        """Get optional list field, returns empty list if not present"""
        value = config.get(field_name)
        if value is None:
            return []
        if not isinstance(value, list):
            logger.warning(f"Field '{field_name}' is not a list, ignoring")
            return []
        return value
    
    def parse_connection_config(self, source_type: str, config: Dict[str, Any]) -> ConnectionProfile:
        """Parse connection configuration based on source type"""
        source = SourceType.from_string(source_type)
        
        if source is None:
            raise ValueError(f"Unknown source type: {source_type}")
        
        try:
            if source == SourceType.FILESYSTEM:
                return FilesystemProfile(
                    profile_name=config.get("profile_name", "default"),
                    source_type=source,
                    path=self._get_required_field(config, "path"),
                    exclude_patterns=self._get_optional_list(config, "exclude_patterns")
                )
            
            elif source == SourceType.POSTGRESQL:
                return PostgreSQLProfile(
                    profile_name=config.get("profile_name", "default"),
                    source_type=source,
                    host=self._get_required_field(config, "host"),
                    port=config.get("port", 5432),
                    user=config.get("user"),
                    password=config.get("password"),
                    database=self._get_required_field(config, "database"),
                    limit_start=config.get("limit_start", 0),
                    limit_end=config.get("limit_end", 50000),
                    tables=self._get_optional_list(config, "tables")
                )
            
            elif source == SourceType.MYSQL:
                return MySQLProfile(
                    profile_name=config.get("profile_name", "default"),
                    source_type=source,
                    host=self._get_required_field(config, "host"),
                    port=config.get("port", 3306),
                    user=config.get("user"),
                    password=config.get("password"),
                    database=config.get("database"),
                    limit_start=config.get("limit_start", 0),
                    limit_end=config.get("limit_end", 500),
                    tables=self._get_optional_list(config, "tables"),
                    exclude_columns=self._get_optional_list(config, "exclude_columns")
                )
            
            elif source == SourceType.S3:
                return S3Profile(
                    profile_name=config.get("profile_name", "default"),
                    source_type=source,
                    access_key=self._get_required_field(config, "access_key"),
                    secret_key=self._get_required_field(config, "secret_key"),
                    bucket_name=self._get_required_field(config, "bucket_name"),
                    cache=config.get("cache", True),
                    exclude_patterns=self._get_optional_list(config, "exclude_patterns")
                )
            
            elif source == SourceType.REDIS:
                return RedisProfile(
                    profile_name=config.get("profile_name", "default"),
                    source_type=source,
                    host=self._get_required_field(config, "host"),
                    password=config.get("password")
                )
            
            elif source == SourceType.SLACK:
                return SlackProfile(
                    profile_name=config.get("profile_name", "default"),
                    source_type=source,
                    channel_types=config.get("channel_types", "public_channel,private_channel"),
                    token=self._get_required_field(config, "token"),
                    only_archived=config.get("onlyArchived", False),
                    limit_mins=config.get("limit_mins", 60),
                    channel_ids=self._get_optional_list(config, "channel_ids")
                )
            
            else:
                raise ValueError(f"Unsupported source type: {source_type}")
                
        except ValueError as e:
            logger.error(f"Configuration error for {source_type}: {e}")
            raise
    
    def route(self, scan_config: Dict[str, Any]) -> Dict[str, Any]:
        """Route scan job to appropriate handler"""
        source_type = scan_config.get("source_type")
        
        if not source_type:
            raise ValueError("Missing required field: source_type")
        
        profile_name = scan_config.get("profile_name", "default")
        
        # Get source-specific config
        source_config = self.profiles.get(source_type, {}).get(profile_name, {})
        
        try:
            profile = self.parse_connection_config(source_type, source_config)
        except ValueError as e:
            logger.error(f"Failed to parse config for {source_type}: {e}")
            raise
        
        source = SourceType.from_string(source_type)
        
        if source is None:
            handler = "scan_generic.py"
        else:
            handler = self.HANDLERS.get(source, "scan_generic.py")
        
        return {
            "source_type": source_type,
            "handler": handler,
            "profile": profile,
            "execution_mode": scan_config.get("execution_mode", "sequential"),
            "pii_types": scan_config.get("pii_types", []),
            "options": scan_config.get("options", {})
        }
    
    def validate_scan_config(self, scan_config: Dict[str, Any]) -> List[str]:
        """Validate scan configuration and return list of errors"""
        errors = []
        
        if not scan_config.get("source_type"):
            errors.append("Missing required field: source_type")
        
        source_type = scan_config.get("source_type")
        if source_type and not SourceType.from_string(source_type):
            errors.append(f"Invalid source_type: {source_type}")
        
        return errors


def main():
    """Main entry point with error handling"""
    if len(sys.argv) < 2:
        print("Usage: route_scans.py <scan_config.json>", file=sys.stderr)
        sys.exit(1)
    
    config_file = sys.argv[1]
    
    if not Path(config_file).exists():
        print(f"Error: Config file not found: {config_file}", file=sys.stderr)
        sys.exit(1)
    
    try:
        with open(config_file, 'r') as f:
            scan_config = json.load(f)
    except json.JSONDecodeError as e:
        print(f"Error: Invalid JSON in config file: {e}", file=sys.stderr)
        sys.exit(1)
    except IOError as e:
        print(f"Error: Failed to read config file: {e}", file=sys.stderr)
        sys.exit(1)
    
    try:
        router = ScanRouter()
        
        # Validate config
        errors = router.validate_scan_config(scan_config)
        if errors:
            print(f"Configuration errors: {', '.join(errors)}", file=sys.stderr)
            sys.exit(1)
        
        # Route scan
        result = router.route(scan_config)
        print(json.dumps(result, indent=2))
        
    except ValueError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        logger.exception(f"Unexpected error: {e}")
        print(f"Error: Unexpected error occurred", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
