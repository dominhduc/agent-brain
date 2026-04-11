# Topic Taxonomy

Detailed descriptions and examples for each of the 8 knowledge topics.

## ui

**Use for:** Frontend, styling, UX, components, responsive design, accessibility.

**Keywords:** React, Vue, Svelte, CSS, Tailwind, styling, layout, responsive, mobile, accessibility, ARIA, component, form, input, button, modal, navigation, animation, theme, dark mode, light mode, color, typography, font, spacing, grid, flexbox.

**Examples:**
```bash
brain add ui gotcha "Tailwind's safelist must include dynamic class names"
brain add ui pattern "All forms use react-hook-form with Zod validation"
brain add ui "Modal backdrop uses fixed positioning with z-index 50"
brain add ui accessibility "All interactive elements have aria-label or aria-labelledby"
```

## backend

**Use for:** API, business logic, services, handlers, middleware, server-side code.

**Keywords:** API, REST, GraphQL, gRPC, handler, middleware, service, controller, router, endpoint, request, response, validation, authentication, authorization, rate limiting, caching, background job, queue, worker, WebSocket.

**Examples:**
```bash
brain add backend gotcha "Express middleware order matters: cors -> body-parser -> auth -> routes"
brain add backend pattern "All handlers return consistent error format: { error, code, details }"
brain add backend "User service uses dependency injection for testability"
brain add backend performance "Database queries use connection pooling with max 20 connections"
```

## infrastructure

**Use for:** Deployment, VPS, CI/CD, Docker, cloud, monitoring, networking.

**Keywords:** Docker, Kubernetes, deployment, CI/CD, GitHub Actions, Jenkins, VPS, AWS, GCP, Azure, Nginx, load balancer, CDN, monitoring, logging, alerting, Prometheus, Grafana, Terraform, Ansible, scaling, autoscaling, backup, disaster recovery.

**Examples:**
```bash
brain add infrastructure gotcha "Docker build cache is invalidated by COPY before package.json"
brain add infrastructure pattern "GitHub Actions uses reusable workflows for deploy pipelines"
brain add infrastructure "Production runs on AWS ECS Fargate with 2 replicas minimum"
brain add infrastructure monitoring "Prometheus scrapes /metrics every 15s; retention is 30 days"
```

## database

**Use for:** Schemas, migrations, queries, indexes, ORM, data modeling.

**Keywords:** PostgreSQL, MySQL, MongoDB, Redis, schema, migration, query, index, foreign key, primary key, constraint, transaction, ACID, ORM, Prisma, TypeORM, Sequelize, SQLAlchemy, data model, normalization, denormalization, replication, sharding.

**Examples:**
```bash
brain add database gotcha "PostgreSQL jsonb columns can't use standard equality indexes"
brain add database pattern "All tables have created_at and updated_at timestamps"
brain add database "User sessions stored in Redis with 7-day TTL"
brain add database performance "Composite index on (user_id, created_at DESC) for feed queries"
```

## security

**Use for:** Auth, secrets, permissions, OWASP, encryption, input validation.

**Keywords:** authentication, authorization, JWT, OAuth, session, cookie, CSRF, XSS, SQL injection, OWASP, encryption, hashing, bcrypt, salt, API key, secret, environment variable, HTTPS, TLS, CORS, rate limiting, input validation, sanitization.

**Examples:**
```bash
brain add security gotcha "JWT tokens must be rotated before expiry; use refresh tokens"
brain add security pattern "All API endpoints validate input with Zod schemas"
brain add security "CORS allows only *.example.com origins in production"
brain add security secrets "API keys stored in AWS Secrets Manager, never in .env"
```

## testing

**Use for:** Unit tests, integration tests, e2e, mocks, fixtures, test runners.

**Keywords:** Jest, Vitest, pytest, Playwright, Cypress, unit test, integration test, e2e test, mock, stub, fixture, factory, test data, coverage, snapshot, TDD, test-driven, assertion, beforeEach, afterEach, setup, teardown.

**Examples:**
```bash
brain add testing gotcha "Jest timers must be faked with jest.useFakeTimers() for date-based tests"
brain add testing pattern "All API tests use a test database with automatic rollback"
brain add testing "E2E tests run on every PR via GitHub Actions with Playwright"
brain add testing mocks "External API calls are mocked with msw (Mock Service Worker)"
```

## architecture

**Use for:** Module structure, design patterns, data flow, system design.

**Keywords:** architecture, design pattern, module, package, layer, tier, monolith, microservice, serverless, event-driven, CQRS, domain-driven design, clean architecture, hexagonal, MVC, MVVM, repository, service layer, dependency injection, separation of concerns.

**Examples:**
```bash
brain add architecture gotcha "Circular dependencies between packages cause build failures"
brain add architecture pattern "Repository pattern abstracts database access from services"
brain add architecture "User module is a bounded context with its own database schema"
brain add architecture data-flow "Events flow: API -> Command Handler -> Aggregate -> Event Store -> Projection"
```

## general

**Use for:** Cross-cutting knowledge, project-wide conventions, tooling.

**Keywords:** convention, standard, guideline, best practice, code style, linting, formatting, git workflow, branch strategy, commit message, code review, documentation, README, changelog, versioning, semantic versioning, package manager, build tool, development environment.

**Examples:**
```bash
brain add general gotcha "Commit messages must follow Conventional Commits format"
brain add general pattern "All PRs require two approvals and passing CI"
brain general "Development requires Node 18+ and pnpm as package manager"
brain add general tooling "Prettier formats on save; ESLint runs on pre-commit hook"
```

## Cross-Topic Entries

Some entries span multiple topics. Use compound tags:

```bash
brain add security database "Database connections use SSL with certificate validation"
brain add backend security "API keys are validated via middleware before route handlers"
brain add infrastructure security "All production traffic goes through Cloudflare WAF"
brain add ui testing "Component tests use React Testing Library with accessibility assertions"
```

## Topic Selection Guidelines

1. **Pick the most specific topic** — `database` is better than `backend` for schema changes
2. **Use compound tags for cross-cutting concerns** — `security database` for encrypted columns
3. **Default to `general` for project-wide conventions** — git workflow, code style, tooling
4. **Use `architecture` for structural decisions** — module boundaries, design patterns
5. **Use `infrastructure` for anything deployment-related** — CI/CD, Docker, cloud, monitoring
