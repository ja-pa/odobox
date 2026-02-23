Technical Specification
Adding Optional HTTP Backend API to a Wails (Go + React) Application
1. Objective

Introduce an optional HTTP backend API layer to an existing Wails application (Go backend + React frontend) without requiring a full architectural rewrite.

The API must:

Enable easier debugging and development

Allow external testing (curl/Postman/integration tests)

Preserve existing Wails JS→Go bindings

Avoid breaking production builds

Maintain security by default

2. Non-Goals

No migration to a pure web server architecture

No replacement of Wails bindings

No public-facing API exposure

No requirement to separate the app into multiple deployable services

3. High-Level Architecture
Current Architecture

React (frontend)
↕ (Wails bindings)
Go backend logic

Target Architecture

React (frontend)
↕ (Wails bindings)
Thin Wails layer
↕
Core business logic (new shared layer)
↕
Optional HTTP API layer (dev/test use)

Key principle:

Business logic must not depend on Wails.

4. Required Refactoring
4.1 Extract Core Business Logic

Create a new package:

internal/core

or

pkg/core

Responsibilities:

Contain pure business logic

No Wails imports

No HTTP-specific dependencies

Deterministic and testable

Example structure:

internal/
  core/
    service.go
    models.go
    errors.go
4.2 Wails Layer Refactor

Wails-bound methods must:

Validate input

Call core functions

Return structured responses

Contain no business logic

Wails becomes an adapter, not a logic container.

5. HTTP API Layer
5.1 Purpose

Development debugging

Integration testing

Optional automation entrypoint

5.2 Scope

Must be optional

Must not run in production unless explicitly enabled

Must bind to 127.0.0.1 only

5.3 Server Implementation

Use net/http or lightweight router (chi recommended)

Start server inside OnStartup(ctx)

Shutdown gracefully on application exit

Example lifecycle:

Application starts

If ENABLE_HTTP_API=true

Start HTTP server on configured port

On shutdown

Gracefully close server

5.4 Configuration

Configuration via:

Environment variables

Config file

Build flags

Required config options:

Variable	Description
ENABLE_HTTP_API	true/false
HTTP_API_PORT	Port number
HTTP_API_TOKEN	Optional security token
6. API Security Requirements

Minimum security requirements:

Bind to 127.0.0.1

No external exposure

Optional static or runtime-generated token

No arbitrary command execution endpoints

No filesystem access without strict control

If future external exposure is required:

Add authentication layer

Add rate limiting

Add audit logging

7. Frontend Integration Strategy

Two modes:

Development Mode

React may call:

http://localhost:<port>/api/...

This enables:

Network inspection

Independent backend testing

Faster debugging

Production Mode

React continues using:

Wails bindings

Switching strategy can be:

Environment flag

Build-time flag

Runtime detection

8. Testing Strategy
Unit Tests

Target core package only

No Wails

No HTTP

Integration Tests

Start HTTP server

Test endpoints using HTTP client

Manual Debug

Use curl or Postman

Validate JSON responses

9. Risks
Risk	Mitigation
Logic duplicated between Wails and HTTP	Centralize in core
Security exposure	Bind to localhost only
Architecture drift toward web app	Keep HTTP optional
Overengineering	Scope control
10. Decision Matrix
Use Case	Recommendation
Debugging only	Optional localhost API
Automated tests	Strongly recommended
External integrations	Requires security hardening
Full SaaS backend	Separate service recommended
11. Acceptance Criteria

Core logic exists in independent package

Wails layer contains no business logic

HTTP server can be enabled/disabled via config

Application works unchanged when API disabled

Core logic covered by unit tests

API endpoints return structured JSON

12. Strategic Positioning

This change:

Increases testability

Improves maintainability

Reduces frontend/backend coupling

Prepares system for future expansion

It does not convert the app into a web service.
