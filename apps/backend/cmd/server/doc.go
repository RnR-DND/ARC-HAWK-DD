// Package main ARC-HAWK-DD API
//
// @title ARC-HAWK Data Discovery & Governance API
// @version 3.0.0
// @description PII discovery, compliance enforcement, and data governance platform. Scans databases, cloud storage, files, and SaaS systems for sensitive data, maps lineage, and automates remediation.
//
// @contact.name ARC-HAWK Support
// @contact.email support@arc-hawk.io
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:8080
// @BasePath /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer token — prefix with "Bearer "
//
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description Pre-provisioned per-tenant API key
//
// @securityDefinitions.apikey ScannerToken
// @in header
// @name X-Scanner-Token
// @description Internal scanner callback token (SCANNER_SERVICE_TOKEN env var)
package main
