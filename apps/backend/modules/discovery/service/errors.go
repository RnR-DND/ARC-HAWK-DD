package service

import "errors"

// ErrSnapshotNotFound is returned when a snapshot ID does not exist for the tenant.
var ErrSnapshotNotFound = errors.New("discovery: snapshot not found")

// ErrSnapshotInProgress is returned when a snapshot is already running for the tenant.
var ErrSnapshotInProgress = errors.New("discovery: snapshot already in progress for tenant")

// ErrReportNotFound is returned when a report ID does not exist for the tenant.
var ErrReportNotFound = errors.New("discovery: report not found")

// ErrReportNotReady is returned when a report download is requested before generation completes.
var ErrReportNotReady = errors.New("discovery: report not ready")

// ErrUpstreamUnavailable is returned when an upstream module (assets, scanning, lineage) errors.
var ErrUpstreamUnavailable = errors.New("discovery: upstream unavailable")
