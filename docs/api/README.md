# O.W.A.S.A.K.A. SIEM - API Documentation

## Overview

This directory will contain API documentation for the O.W.A.S.A.K.A. SIEM platform.

**Status**: PHASE 0 - Foundation (API not yet implemented)

---

## Planned API Endpoints

### Authentication
- `POST /auth/login` - Authenticate user (future)
- `POST /auth/logout` - Logout user (future)
- `POST /auth/refresh` - Refresh authentication token (future)

### Assets
- `GET /api/v1/assets` - List all assets
- `GET /api/v1/assets/:id` - Get specific asset
- `POST /api/v1/assets` - Create asset (manual)
- `PUT /api/v1/assets/:id` - Update asset
- `DELETE /api/v1/assets/:id` - Delete asset

### Network Events
- `GET /api/v1/events/network` - List network events
- `GET /api/v1/events/network/:id` - Get specific event
- `GET /api/v1/events/network/stream` - WebSocket stream

### DNS Events
- `GET /api/v1/events/dns` - List DNS events
- `GET /api/v1/events/dns/:id` - Get specific DNS event
- `GET /api/v1/events/dns/stream` - WebSocket stream

### Alerts
- `GET /api/v1/alerts` - List alerts
- `GET /api/v1/alerts/:id` - Get specific alert
- `PUT /api/v1/alerts/:id` - Update alert (assign, notes, status)
- `POST /api/v1/alerts/:id/notes` - Add note to alert
- `GET /api/v1/alerts/stream` - WebSocket stream

### Services
- `GET /api/v1/services` - List discovered services
- `GET /api/v1/services/:id` - Get specific service
- `GET /api/v1/services/asset/:asset_id` - List services for asset

### Topology
- `GET /api/v1/topology` - Get network topology graph
- `GET /api/v1/topology/stream` - WebSocket stream for updates

### Discovery
- `POST /api/v1/discovery/scan` - Trigger manual scan
- `GET /api/v1/discovery/status` - Get scan status
- `GET /api/v1/discovery/history` - Get scan history

### Analytics
- `GET /api/v1/analytics/dashboard` - Get dashboard metrics
- `GET /api/v1/analytics/risk-score` - Get overall risk score
- `GET /api/v1/analytics/trends` - Get security trends

### Configuration (Admin)
- `GET /api/v1/config` - Get current configuration
- `PUT /api/v1/config` - Update configuration (requires restart)

### Health & Metrics
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics
- `GET /api/v1/status` - System status

---

## WebSocket API

### Connection
```
ws://localhost:8080/ws
```

### Message Format
```json
{
  "type": "subscribe",
  "channel": "events.network",
  "filters": {
    "asset_id": "uuid-here",
    "severity": "high"
  }
}
```

### Channels
- `events.network` - Network events
- `events.dns` - DNS events
- `alerts` - Security alerts
- `topology` - Topology updates
- `discovery` - Discovery scan updates

---

## Response Format

### Success Response
```json
{
  "data": {
    "id": "uuid",
    "type": "asset",
    "attributes": { }
  },
  "meta": {
    "timestamp": "2025-10-25T10:30:00Z",
    "query_time_ms": 45
  }
}
```

### Error Response
```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid asset ID format",
    "details": {
      "field": "asset_id",
      "reason": "must be a valid UUID"
    }
  },
  "timestamp": "2025-10-25T10:30:00Z"
}
```

### Paginated Response
```json
{
  "data": [ ],
  "pagination": {
    "page": 1,
    "per_page": 50,
    "total": 1234,
    "total_pages": 25,
    "has_next": true,
    "has_prev": false
  },
  "meta": {
    "timestamp": "2025-10-25T10:30:00Z"
  }
}
```

---

## Authentication (Future)

### mTLS (Recommended)
Client certificates for mutual TLS authentication.

### JWT (Alternative)
Bearer token authentication with configurable expiry.

---

**Document Version**: 0.1.0
**Last Updated**: 2025-10-25
**Status**: PHASE 0 - Planned, not implemented
