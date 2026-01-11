# Zagalin Version History

A comprehensive overview of Zagalin's evolution, organized by release theme and major milestones.

---

## Release Roadmap

### Current Version: 0.0.2 (December 27, 2025)

### Next Version: 0.0.3 (Expected: January 2026)

### Future: 1.0.0 (Expected: Q1 2026)

---

## Version 0.0.2 - "Security & Governance" (December 27, 2025)

### Theme

Production-ready security controls for enterprise deployments.

### Headline Features

1. **Query Validation System** - Pattern-based validation for PromQL, LogQL, TraceQL
2. **OpenTelemetry Scope Enforcement** - Automatic service/environment labeling
3. **Datasource Governance** - Allowlist system for approved datasources
4. **Conversation History** - Persistent storage with dual-tier architecture
5. **AI Development Tools** - Configurations for Claude, ChatGPT, Copilot, Cursor
6. **Privacy-Conscious Logging** - Usage analytics without exposing queries

### Key Metrics

- **Code Changes**: +3,000 lines
- **Security Pipeline Stages**: 6
- **Configuration Options**: 25+ (all opt-in)
- **Test Coverage**: >90% for query validation
- **Breaking Changes**: None

### Target Users

- Enterprise teams with compliance requirements
- Organizations enforcing OpenTelemetry standards
- Multi-tenant Grafana deployments
- Security-conscious teams

### Documentation

- [Release Notes](releases/v0.0.2.md)
- [Detailed Changelog](../CHANGELOG.md#002---2025-12-27)
- [Upgrade Guide](../CHANGELOG.md#upgrading-to-002-from-001)

---

## Version 0.0.1 - "Foundation" (December 24, 2025)

### Theme

Initial release establishing core AI assistant capabilities.

### Headline Features

1. **Context-Aware Chat** - Dashboard, panel, and time range awareness
2. **Floating Chat Interface** - Global chat button on every dashboard
3. **Query Generation** - Natural language to PromQL/LogQL/TraceQL
4. **Skills System** - Auto-detected user intent (explain, generate, troubleshoot, analyze)
5. **LLM Integration** - Provider-agnostic support via grafana-llm-app
6. **Function Calling** - Structured tool execution (navigate, create query, open explore)

### Key Metrics

- **Initial Codebase**: ~8,000 lines
- **Supported Query Languages**: 3 (PromQL, LogQL, TraceQL)
- **Skills**: 4 (explain_panel, generate_query, troubleshoot, analyze_dashboard)
- **Minimum Grafana Version**: 10.4.0
- **Breaking Changes**: N/A (initial release)

### Target Users

- SREs and platform engineers
- DevOps teams using Grafana
- Observability practitioners
- Early adopters of AI-assisted workflows

### Documentation

- [Initial README](../README.md)
- [Changelog](../CHANGELOG.md#001---2025-12-24)

---

## Feature Evolution Matrix

Track when major features were introduced:

| Feature                | v0.0.1 | v0.0.2      | v0.0.3 (planned) |
| ---------------------- | ------ | ----------- | ---------------- |
| **Core**               |
| Context-aware chat     |        |             |                  |
| Floating chat UI       |        |             |                  |
| Query generation       |        |             |                  |
| Skills system          |        |             | (enhanced)       |
| **Security**           |
| Query validation       |        |             |                  |
| OTel enforcement       |        |             |                  |
| Datasource governance  |        |             |                  |
| Audit logging          |        |             |                  |
| **Storage**            |
| Conversation history   |        |             |                  |
| Export (JSON/Markdown) |        | Placeholder |                  |
| **Orchestration**      |
| Simple streaming       |        |             |                  |
| Frontend orchestration |        | In progress |                  |
| Planning & steps       |        | In progress |                  |
| Artifact extraction    |        | In progress |                  |
| **Dev Experience**     |
| AI dev tools           |        |             |                  |
| Local CI pipeline      |        |             |                  |
| Pre-commit hooks       |        |             |                  |

Legend: Stable | Beta/Partial | Not Available

---

## Version Themes Timeline

```
2025-12-24: v0.0.1 "Foundation"
     Core AI assistant capabilities
         Context awareness, query generation, floating UI

2025-12-27: v0.0.2 "Security & Governance"
     Production-ready security controls
         Query validation, OTel enforcement, governance

2026-01 (planned): v0.0.3 "Intelligence & Orchestration"
     Structured investigation workflows
         Planning, step execution, artifacts, smart routing

2026-Q1 (planned): v1.0.0 "Production Maturity"
     Feature-complete, battle-tested, documented
         Performance optimizations, advanced features
```

---

## Breaking Changes History

### v0.0.2

**None**. All new features are opt-in and disabled by default.

### v0.0.1

**N/A** (initial release)

---

## Configuration Evolution

### v0.0.1

**Required Configuration**:

```json
{
  "llmBackend": "grafana-llm-app" // Only required field
}
```

### v0.0.2

**Required Configuration**: Same as v0.0.1

**New Optional Configuration** (25+ new fields):

- Query validation settings (7 fields)
- OTel enforcement settings (7 fields)
- Datasource governance (2 fields)
- Usage logging (1 field)
- Conversation storage (auto-configured)

**Backward Compatibility**: 100% - All new settings are disabled by default

---

## Deprecation Timeline

No deprecations yet. All features introduced in v0.0.1 are still active and maintained.

---

## Performance Evolution

### Query Validation

- **v0.0.2**: Pattern-based matching, <1ms overhead per query
- **Future**: Potential AST parsing for advanced validation

### Context Extraction

- **v0.0.1**: Basic dashboard context, ~50-200ms
- **v0.0.2**: Same performance, added conversation context
- **v0.0.3 (planned)**: Optimized context, ~20-50ms

### LLM Response Time

- **v0.0.1**: Streaming starts in ~500-1000ms (depends on LLM provider)
- **v0.0.2**: Same (no changes to LLM integration)
- **v0.0.3 (planned)**: Planning phase adds ~2-3s upfront, better structured output

---

## Security Evolution

### v0.0.1

- Session-based authentication (inherited from Grafana)
- No credential storage
- XSS protection via DOMPurify
- Basic rate limiting (60 req/min)

### v0.0.2

- **Added**: 6-step security pipeline
- **Added**: Query injection prevention (SQL, PromQL, LogQL, TraceQL)
- **Added**: Query complexity limits
- **Added**: Datasource access control
- **Added**: OpenTelemetry scope enforcement
- **Added**: Comprehensive audit logging with user context
- **Added**: Privacy-conscious usage logging

---

## Adoption Recommendations

### For Early Adopters (v0.0.1 → v0.0.2)

1. Start with **validation-only mode** to understand impact
2. Enable **conversation history** immediately (no downside)
3. Configure **OTel enforcement** after understanding defaults
4. Add **datasource governance** when ready for stricter controls

### For New Users (v0.0.2+)

1. Start with **default configuration** (all security features disabled)
2. Enable features gradually based on needs:
   - Day 1: Basic LLM integration only
   - Week 1: Enable conversation history
   - Week 2: Enable query validation in validation-only mode
   - Week 3: Enable strict mode + OTel enforcement
   - Week 4: Add datasource governance

### For Enterprise (v0.0.2+)

1. **Mandatory**: Enable all security features in strict mode from day 1
2. **Recommended**: Start with production-recommended settings
3. **Required**: Set up audit log monitoring
4. **Optional**: Customize function allowlists for your environment

---

## Version Comparison Table

| Aspect                   | v0.0.1         | v0.0.2                | v0.0.3 (planned)             |
| ------------------------ | -------------- | --------------------- | ---------------------------- |
| **Release Date**         | Dec 24, 2025   | Dec 27, 2025          | Jan 2026                     |
| **Theme**                | Foundation     | Security & Governance | Intelligence & Orchestration |
| **Codebase Size**        | ~8,000 lines   | ~11,000 lines         | ~14,000 lines (est)          |
| **Security Features**    | Basic (4)      | Advanced (10)         | Advanced (10)                |
| **Configuration Fields** | ~10            | ~35                   | ~45 (est)                    |
| **Test Coverage**        | ~80%           | ~85%                  | ~90% (goal)                  |
| **Breaking Changes**     | N/A            | None                  | TBD                          |
| **Migration Effort**     | N/A            | Zero config           | Low (new features)           |
| **Recommended For**      | Early adopters | Production use        | Power users                  |

---

## Support Matrix

| Zagalin Version | Grafana Versions | grafana-llm-app | Node.js | Go    | Status    |
| --------------- | ---------------- | --------------- | ------- | ----- | --------- |
| 0.0.2           | 10.4.0 - 12.x    | 1.0.0+          | 22+     | 1.21+ | Current   |
| 0.0.1           | 10.4.0 - 12.x    | 1.0.0+          | 22+     | 1.21+ | Supported |

**Support Policy**:

- Latest version: Full support (bug fixes, features)
- Previous version: Security updates only
- Older versions: No support (please upgrade)

---

## Documentation Evolution

### v0.0.1

- README.md
- Basic architecture overview
- Development setup guide

### v0.0.2

- **Added**: Comprehensive CHANGELOG.md (Keep a Changelog format)
- **Added**: Release notes (docs/releases/v0.0.2.md)
- **Added**: Version history (this document)
- **Added**: GitHub release template
- **Added**: AI development tools documentation
- **Enhanced**: .claude/CLAUDE.md (500+ lines)
- **Enhanced**: API documentation
- **Enhanced**: Configuration guides

### v0.0.3 (planned)

- Feature-specific guides (orchestration, artifacts)
- Video tutorials
- Interactive examples
- Architecture diagrams

---

## Statistics & Metrics

### Codebase Growth

- **v0.0.1**: ~8,000 lines (initial)
- **v0.0.2**: ~11,000 lines (+37.5%)
- **v0.0.3 (est)**: ~14,000 lines (+27%)

### Feature Count

- **v0.0.1**: 10 major features
- **v0.0.2**: 16 major features (+60%)
- **v0.0.3 (est)**: 22 major features (+37.5%)

### Test Coverage

- **v0.0.1**: ~80%
- **v0.0.2**: ~85% (+5%)
- **v0.0.3 (goal)**: ~90% (+5%)

---

## Links

- **Changelog**: [CHANGELOG.md](../CHANGELOG.md)
- **Releases**: [docs/releases/](releases/)
- **GitHub**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app
- **Issues**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues
- **Grafana Plugin Catalog**: https://grafana.com/grafana/plugins/jorgeancal-zagalin-app
