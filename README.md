# CodeShop

<p align="center">
  <strong>Production-Oriented Digital Marketplace built with Go, Microservices, Kubernetes, DevOps & DevSecOps</strong>
</p>

<p align="center">
  A digital marketplace for selling source code, software templates, SaaS starters, and other digital products with secure payment and controlled downloads.
</p>

<p align="center">

![Go](https://img.shields.io/badge/Go-1.x-00ADD8?logo=go&logoColor=white)
![gRPC](https://img.shields.io/badge/gRPC-API-244c5a?logo=google)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Database-4169E1?logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-Cache-DC382D?logo=redis&logoColor=white)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-Messaging-FF6600?logo=rabbitmq&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Container-2496ED?logo=docker&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Orchestration-326CE5?logo=kubernetes&logoColor=white)
![GitHub Actions](https://img.shields.io/badge/GitHub_Actions-CI-2088FF?logo=github-actions&logoColor=white)
![Argo CD](https://img.shields.io/badge/Argo_CD-GitOps-EF7B4D?logo=argo&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-Monitoring-E6522C?logo=prometheus&logoColor=white)
![Grafana](https://img.shields.io/badge/Grafana-Observability-F46800?logo=grafana&logoColor=white)
![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Tracing-7B42BC?logo=opentelemetry&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

</p>

---

## Overview

**CodeShop** is a production-oriented digital marketplace designed for selling and securely delivering digital software products.

The platform can be used to sell:

- Source code
- Laravel templates
- Go projects
- Next.js templates
- SaaS starter kits
- Admin dashboards
- UI templates
- API templates
- Developer tools
- Other downloadable digital products

The system is designed using **microservices architecture** and will progressively evolve from local development into a Kubernetes-based production environment.

The project focuses not only on application development, but also on the complete engineering lifecycle:

```text
Architecture
     ↓
Development
     ↓
Testing
     ↓
Containerization
     ↓
CI
     ↓
Security Scanning
     ↓
Container Registry
     ↓
Kubernetes
     ↓
GitOps
     ↓
Staging
     ↓
Production
     ↓
Observability
     ↓
Security Operations
     ↓
Disaster Recovery
```

---

# Why CodeShop?

CodeShop is designed as more than a simple e-commerce application.

The project is also a practical implementation of modern:

- Backend Engineering
- Distributed Systems
- Microservices Architecture
- DevOps
- Cloud Engineering
- Kubernetes
- CI/CD
- GitOps
- DevSecOps
- Observability
- SRE
- Disaster Recovery

The architecture is intentionally designed to demonstrate how a system can evolve from a local development environment into a production-oriented platform.

---

# Core Features

## Digital Marketplace

- Product catalog
- Product categories
- Product versions
- Product details
- Product search
- Product management
- Digital product delivery

## Authentication

- User registration
- Login
- JWT authentication
- Refresh token
- Role-based access control
- User management

## Shopping

- Product browsing
- Shopping cart
- Checkout
- Order creation
- Order history
- Order status

## Payment

- Payment initialization
- Payment provider integration
- Payment verification
- Secure webhook processing
- Payment state management
- Idempotent payment handling
- Payment reconciliation

## Secure Download

Purchased products are stored inside **private object storage**.

Users cannot directly access the original files.

The download process is:

```text
User
  │
  ▼
API Gateway
  │
  ▼
Download Service
  │
  ├── Authenticate
  ├── Authorize
  ├── Verify Entitlement
  ├── Verify Order / Payment
  └── Generate Short-Lived Signed URL
               │
               ▼
       Private Object Storage
```

This prevents digital products from being exposed through permanent public URLs.

> A valid signed URL can technically be shared while it remains valid. CodeShop mitigates this risk through short expiration times, authorization checks, rate limiting, download policies, and audit logging.

---

# Architecture

CodeShop uses a microservices architecture with clear domain ownership.

```text
                         ┌─────────────────────┐
                         │       Client        │
                         │   Web / Mobile App  │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │     Cloudflare      │
                         │     / Ingress       │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │     API Gateway     │
                         └──────────┬──────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              │                     │                     │
              ▼                     ▼                     ▼
       ┌─────────────┐       ┌─────────────┐       ┌─────────────┐
       │    Auth     │       │   Catalog   │       │    Order    │
       │   Service   │       │   Service   │       │   Service   │
       └──────┬──────┘       └──────┬──────┘       └──────┬──────┘
              │                     │                     │
              ▼                     ▼                     ▼
         PostgreSQL           PostgreSQL              PostgreSQL

                                    │
                                    ▼
                             ┌─────────────┐
                             │   Payment   │
                             │   Service   │
                             └──────┬──────┘
                                    │
                                    ▼
                            Payment Provider

                                    │
                                    ▼
                               ┌─────────┐
                               │ RabbitMQ│
                               └────┬────┘
                                    │
                       ┌────────────┼────────────┐
                       │            │            │
                       ▼            ▼            ▼
                Download       Notification   Other
                 Service         Service      Consumers
                    │
                    ▼
            Private Object Storage
```

---

# Microservices

| Service | Responsibility |
|---|---|
| API Gateway | Routing, rate limiting, request handling and edge concerns |
| Auth Service | Authentication, identity and access control |
| Catalog Service | Products, categories and product metadata |
| Order Service | Cart, checkout and order lifecycle |
| Payment Service | Payment transactions and provider integration |
| Download Service | Download entitlement and secure digital delivery |
| Notification Service | User and system notifications |

Each service owns its own domain and data.

The architecture follows:

> **Database per Service**

A service must never directly access another service's database.

---

# Service Communication

CodeShop uses two primary communication patterns.

## Synchronous Communication

Used when an immediate response is required.

```text
Client
  │
  ▼
API Gateway
  │
  ▼
Service
```

Public APIs use:

```text
REST / HTTP
```

Internal service communication may use:

```text
gRPC
```

## Asynchronous Communication

Used for domain events and decoupled workflows.

```text
Service
   │
   ▼
RabbitMQ
   │
   ├── Consumer A
   ├── Consumer B
   └── Consumer C
```

Potential domain events include:

```text
UserRegistered
OrderCreated
PaymentSucceeded
OrderPaid
DownloadGranted
DownloadCompleted
RefundCompleted
```

---

# Payment Architecture

Payment and Order are separate domains.

## Payment Service owns

```text
Payment
PaymentTransaction
PaymentStatus
ProviderReference
WebhookProcessing
```

## Order Service owns

```text
Order
OrderItem
OrderStatus
OrderLifecycle
```

The canonical payment flow is:

```text
Payment Provider
       │
       ▼
Payment Service
       │
       ├── Verify Webhook
       ├── Validate Amount
       ├── Validate Reference
       └── Idempotency Check
       │
       ▼
PaymentSucceeded
       │
       ▼
RabbitMQ
       │
       ▼
Order Service
       │
       ▼
OrderPaid
       │
       ├───────────────┐
       ▼               ▼
Download        Notification
Service            Service
```

This keeps payment and order ownership clearly separated.

---

# Security

Security is treated as a first-class architecture concern.

## Authentication

The system is designed to support:

- Password hashing
- JWT
- Refresh tokens
- Token expiration
- Role-based access control
- Least privilege
- Secure session/token management

## Authorization

The API Gateway is **not** the ultimate business authorization authority.

Business authorization remains inside the service that owns the domain.

Example:

```text
User
 ↓
API Gateway
 ↓
Download Service
 ↓
Authentication
 ↓
Entitlement Check
 ↓
Order Check
 ↓
Payment Check
 ↓
Allow Download
```

---

# Payment Webhook Security

Payment-provider webhooks use a dedicated security model.

They do not rely on normal user JWT authentication.

```text
Payment Provider
       │
       ▼
Webhook Endpoint
       │
       ├── Signature Verification
       ├── Schema Validation
       ├── Merchant Validation
       ├── Amount Validation
       ├── Replay Protection
       └── Idempotency Check
              │
              ▼
        Persist Payment
              │
              ▼
        Publish Event
```

This protects the payment flow against:

- Fake webhooks
- Replay attacks
- Duplicate webhooks
- Amount manipulation
- Payment/order mismatch

---

# DevSecOps

Security is integrated throughout the software lifecycle.

Target pipeline:

```text
Developer
    │
    ▼
Pull Request
    │
    ├── Lint
    ├── Unit Tests
    ├── Integration Tests
    ├── SAST
    ├── Dependency Scan
    ├── Build
    ├── Container Scan
    └── SBOM
          │
          ▼
   Container Registry
          │
          ▼
       GitOps
          │
          ▼
       Argo CD
          │
          ▼
      Kubernetes
```

Security tooling is planned around:

- Trivy
- Syft
- Cosign
- SBOM
- Image signing
- Dependency scanning
- Secret management
- Kubernetes RBAC
- NetworkPolicy
- Least privilege

---

# Technology Stack

## Backend

- Go
- REST
- gRPC

## Database

- PostgreSQL

## Cache

- Redis

## Message Broker

- RabbitMQ

## Object Storage

- MinIO / S3-compatible Object Storage

## Containerization

- Docker

## Orchestration

- Kubernetes
- Helm

## CI/CD

- GitHub Actions

## GitOps

- Argo CD

## Observability

- Prometheus
- Grafana
- Loki
- OpenTelemetry
- Tempo
- Alertmanager

## Security

- Trivy
- Syft
- Cosign
- Kubernetes RBAC
- NetworkPolicy

## Edge / Network

- Cloudflare
- Ingress
- TLS

---

# Repository Structure

```text
codeshop/
│
├── .github/
│   └── workflows/
│
├── services/
│   ├── api-gateway/
│   ├── auth-service/
│   ├── catalog-service/
│   ├── order-service/
│   ├── payment-service/
│   ├── download-service/
│   └── notification-service/
│
├── infrastructure/
│
├── deployments/
│
├── scripts/
│
├── docs/
│   ├── architecture/
│   ├── decisions/
│   └── diagrams/
│
├── README.md
├── LICENSE
├── CHANGELOG.md
├── CONTRIBUTING.md
└── SECURITY.md
```

The repository structure may evolve during implementation.

---

# Development Roadmap

The project is developed progressively.

```text
┌──────────────────────────────────────┐
│ Phase 1                              │
│ Architecture & System Design        │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 2                              │
│ Local Infrastructure                 │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 3                              │
│ Microservice Implementation          │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 4                              │
│ Testing & Quality                    │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 5                              │
│ Continuous Integration               │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 6                              │
│ Kubernetes                           │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 7                              │
│ GitOps & Continuous Delivery         │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 8                              │
│ Observability                        │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 9                              │
│ Security Hardening                   │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 10                             │
│ Production Deployment                │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 11                             │
│ Scaling & Optimization               │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Phase 12                             │
│ Disaster Recovery & Incident Response│
└──────────────────────────────────────┘
```

---

# Environment Strategy

The target environment progression is:

```text
Local
  ↓
Development
  ↓
Staging
  ↓
Production
```

Each environment will have isolated:

- Configuration
- Secrets
- Access control
- Infrastructure
- Monitoring
- Deployment policies

Production must never depend on developer-local configuration.

---

# Kubernetes

Kubernetes is the target production orchestration platform.

The production architecture is designed around:

- Namespaces
- Deployments
- Services
- Ingress
- ConfigMaps
- Secrets
- ServiceAccounts
- RBAC
- NetworkPolicies
- Resource Requests/Limits
- Horizontal Pod Autoscaling
- PodDisruptionBudgets
- Health Probes
- Rolling Deployments
- Graceful Shutdown
- Persistent Storage

Kubernetes is introduced for operational capabilities such as:

- Service orchestration
- Self-healing
- Horizontal scaling
- Declarative infrastructure
- Resource isolation
- Production deployment management

---

# CI/CD & GitOps

The target deployment pipeline is:

```text
                    ┌──────────────┐
                    │   Developer  │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │    GitHub    │
                    └──────┬───────┘
                           │
                           ▼
                  ┌──────────────────┐
                  │ GitHub Actions   │
                  │                  │
                  │ Test             │
                  │ Security Scan    │
                  │ Build            │
                  │ Containerize     │
                  └────────┬─────────┘
                           │
                           ▼
                  ┌──────────────────┐
                  │ Container        │
                  │ Registry         │
                  └────────┬─────────┘
                           │
                           ▼
                  ┌──────────────────┐
                  │ GitOps Repository│
                  └────────┬─────────┘
                           │
                           ▼
                     ┌────────────┐
                     │  Argo CD   │
                     └─────┬──────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Kubernetes  │
                    └──────────────┘
```

The architecture separates:

**Build**

from:

**Deployment**

and provides traceability for production changes.

---

# Observability

CodeShop follows the three pillars of observability:

```text
Metrics
Logs
Traces
```

Target stack:

```text
Services
   │
   ├──────────────┬───────────────┐
   ▼              ▼               ▼
Prometheus       Loki        OpenTelemetry
   │              │               │
   │              │               ▼
   │              │             Tempo
   │              │
   └──────────────┼───────────────┐
                  ▼               │
               Grafana            │
                  │               │
                  ▼               │
             Alertmanager         │
                                  │
                                  ▼
                              Incidents
```

Observability will cover:

### Infrastructure

- CPU
- Memory
- Disk
- Network
- Kubernetes resources

### Application

- Request latency
- Error rate
- Throughput
- Availability

### Infrastructure Services

- PostgreSQL
- Redis
- RabbitMQ
- Object Storage

### Business

- Orders
- Payment success rate
- Payment failure rate
- Downloads
- Download failures

---

# Reliability

The system is designed around distributed-system reliability principles:

- Timeouts
- Selective retries
- Exponential backoff
- Idempotency
- Circuit breakers
- Dead-letter queues
- Graceful shutdown
- Health checks
- Failure isolation
- Eventual consistency
- Partial failure handling

Retries will only be applied where the operation is safe to retry.

---

# Database Architecture

CodeShop follows:

> **Database per Service**

Example:

```text
Auth Service
     │
     ▼
Auth Database

Catalog Service
     │
     ▼
Catalog Database

Order Service
     │
     ▼
Order Database

Payment Service
     │
     ▼
Payment Database

Download Service
     │
     ▼
Download Database
```

No service is allowed to directly query another service's database.

Cross-service communication happens through:

- REST
- gRPC
- Domain Events

---

# Data Protection

Sensitive information is classified based on risk.

| Data | Classification |
|---|---|
| Product metadata | Public / Internal |
| User profile | Sensitive |
| Password hash | Highly Sensitive |
| Access token | Highly Sensitive |
| Payment references | Sensitive |
| Source code | Highly Sensitive |
| Infrastructure secrets | Highly Sensitive |
| Audit logs | Sensitive |

Source-code products are stored in private object storage.

---

# Disaster Recovery

The production architecture includes recovery planning for:

- PostgreSQL
- Object Storage
- RabbitMQ
- Kubernetes configuration
- GitOps configuration
- Application configuration
- Secrets

Recovery lifecycle:

```text
Backup
  ↓
Retention
  ↓
Restore Test
  ↓
Recovery Procedure
  ↓
Validation
```

A backup is not considered reliable until the restore process has been tested.

---

# Incident Response

Production incidents follow a structured lifecycle:

```text
Detection
   ↓
Triage
   ↓
Containment
   ↓
Investigation
   ↓
Recovery
   ↓
Validation
   ↓
Postmortem
   ↓
Preventive Actions
```

Potential incidents include:

- Payment provider outage
- Database outage
- Unauthorized download
- Credential compromise
- Kubernetes failure
- RabbitMQ failure
- Container vulnerability
- Secret leakage

---

# Production Readiness

Before production deployment, the system must satisfy the production readiness requirements.

## Application

- [ ] Authentication
- [ ] Authorization
- [ ] Input validation
- [ ] Error handling
- [ ] Idempotency
- [ ] Graceful shutdown
- [ ] Health checks

## Security

- [ ] TLS
- [ ] Secret management
- [ ] RBAC
- [ ] NetworkPolicy
- [ ] Dependency scanning
- [ ] Container scanning
- [ ] SBOM
- [ ] Image signing
- [ ] Audit logging

## Infrastructure

- [ ] Resource limits
- [ ] Horizontal scaling
- [ ] Pod disruption protection
- [ ] Backup
- [ ] Restore testing
- [ ] Monitoring
- [ ] Alerting

## CI/CD

- [ ] Automated tests
- [ ] Security gates
- [ ] Immutable artifacts
- [ ] GitOps
- [ ] Rollback strategy

## Operations

- [ ] Runbooks
- [ ] Incident response
- [ ] Disaster recovery
- [ ] SLOs
- [ ] RPO/RTO

---

# Scalability Strategy

The system will scale progressively.

```text
Stage 1
Single Kubernetes Cluster
        ↓
Stage 2
Horizontal Service Scaling
        ↓
Stage 3
Caching & Database Optimization
        ↓
Stage 4
Read Replicas / Dedicated Infrastructure
        ↓
Stage 5
Advanced Distributed Architecture
```

Scaling decisions will be based on actual system bottlenecks.

The project intentionally avoids premature optimization.

---

# MVP

The initial MVP focuses on:

- Authentication
- Product catalog
- Product management
- Shopping cart
- Checkout
- Orders
- Payment
- Secure digital downloads
- Basic notifications
- Basic observability

The MVP intentionally uses a **single-seller / platform-owned model**.

Multi-vendor functionality is planned for future expansion.

---

# Future Features

Potential future capabilities include:

- Multi-vendor marketplace
- Seller dashboard
- Seller onboarding
- Commission system
- Seller payouts
- Coupons
- Promotions
- Reviews
- Ratings
- Subscription products
- Software licensing
- Activation keys
- Fraud detection
- Advanced analytics
- Recommendation system
- Search engine
- Multi-region deployment

These features will only be introduced when justified by real requirements.

---

# Documentation

Architecture and engineering documentation can be found under:

```text
docs/
```

Important documentation includes:

```text
docs/
├── architecture/
│   ├── REQUIREMENTS.md
│   ├── ARCHITECTURE.md
│   ├── SERVICE_CATALOG.md
│   ├── DATABASE_DESIGN.md
│   ├── API_DESIGN.md
│   ├── EVENT_DESIGN.md
│   ├── THREAT_MODEL.md
│   ├── KUBERNETES_DESIGN.md
│   ├── CI_CD_DESIGN.md
│   ├── OBSERVABILITY.md
│   ├── RELIABILITY.md
│   ├── DISASTER_RECOVERY.md
│   └── PRODUCTION_READINESS.md
│
├── decisions/
│   └── ADR-*.md
│
└── diagrams/
```

Architecture decisions are documented using **Architecture Decision Records (ADR)**.

---

# Engineering Principles

CodeShop follows these principles:

### Domain Ownership

Every service owns a clearly defined business domain.

### Database Ownership

Every service owns its own data.

### Least Privilege

Every component receives only the permissions it requires.

### Secure by Default

Security is designed into the system from the beginning.

### Observable by Default

Services should provide sufficient telemetry to understand their behavior.

### Failure is Expected

Distributed systems must assume dependencies can fail.

### Automation

Manual operational processes should progressively be automated.

### Immutable Artifacts

Build artifacts should be reproducible, traceable, and immutable.

### Infrastructure as Code

Infrastructure should be declarative and version controlled.

### GitOps

Production configuration should be auditable through Git.

### Progressive Complexity

Complexity should only be introduced when it provides real engineering value.

---

# Project Status

> 🚧 **Active Development**

Current stage:

```text
Phase 1 — Architecture & System Design
```

The architecture is being designed with a long-term production target.

Implementation will proceed incrementally.

---

# Roadmap

| Phase | Description | Status |
|---|---|---|
| 1 | Architecture & System Design | 🟡 In Progress |
| 2 | Local Infrastructure | ⚪ Planned |
| 3 | Microservice Implementation | ⚪ Planned |
| 4 | Testing & Quality | ⚪ Planned |
| 5 | Continuous Integration | ⚪ Planned |
| 6 | Kubernetes | ⚪ Planned |
| 7 | GitOps & Continuous Delivery | ⚪ Planned |
| 8 | Observability | ⚪ Planned |
| 9 | Security Hardening | ⚪ Planned |
| 10 | Production Deployment | ⚪ Planned |
| 11 | Scaling & Optimization | ⚪ Planned |
| 12 | Disaster Recovery & Incident Response | ⚪ Planned |

---

# Portfolio Objectives

CodeShop is built as a practical engineering portfolio demonstrating the complete lifecycle of a modern distributed system.

```text
Software Architecture
        │
        ▼
Microservices
        │
        ▼
Go Backend
        │
        ▼
Docker
        │
        ▼
Testing
        │
        ▼
CI/CD
        │
        ▼
Kubernetes
        │
        ▼
GitOps
        │
        ▼
DevSecOps
        │
        ▼
Observability
        │
        ▼
SRE
        │
        ▼
Production
        │
        ▼
Disaster Recovery
```

The goal is not simply to use as many technologies as possible.

The goal is to demonstrate **why each technology exists, what problem it solves, and how the system behaves under failure and operational pressure.**

---

# Contributing

Contributions, suggestions, architecture discussions, and improvements are welcome.

Please read:

```text
CONTRIBUTING.md
```

before submitting changes.

---

# Security

If you discover a security vulnerability, please follow the instructions in:

```text
SECURITY.md
```

Please avoid publicly disclosing sensitive vulnerabilities before they have been responsibly investigated.

---

# License

This project is licensed under the MIT License.

See:

```text
LICENSE
```

for more information.

---

# Author

**Asep Nurdin**

CodeShop is a long-term engineering project focused on:

- Backend Engineering
- Microservices
- DevOps
- Kubernetes
- DevSecOps
- Cloud Engineering
- Observability
- SRE
- Production Operations

---

<p align="center">
  <strong>CodeShop</strong>
  <br>
  From Digital Marketplace to Production-Oriented Microservices
  <br><br>
  Built with Go • Docker • Kubernetes • GitOps • DevSecOps
</p>
