# Zagalin v{VERSION} - {RELEASE_THEME}

**Release Date**: {DATE}

## 🎯 Release Theme: {RELEASE_THEME}

{HIGH_LEVEL_NARRATIVE - 2-3 paragraphs explaining what this release is about, the problem it solves, and why users should care}

---

## 🚀 What's New

### Major Features

#### {Feature 1 Name}

{2-3 sentence description of what it does and why it matters}

**Key capabilities**:

- Bullet point 1
- Bullet point 2
- Bullet point 3

**Why it matters**: {1-2 sentences on user benefit}

#### {Feature 2 Name}

{Description}

**Key capabilities**:

- Bullet points

**Why it matters**: {User benefit}

### Improvements

- **{Area}**: {Description of improvement}
- **{Area}**: {Description of improvement}

### Bug Fixes

- Fixed {issue description}
- Resolved {issue description}
- Corrected {issue description}

---

## 📊 By the Numbers

- **Features added**: X
- **Bugs fixed**: X
- **Code changes**: +X lines
- **Test coverage**: X%
- **Performance improvement**: X%

---

## 🔧 Breaking Changes

{IF ANY - otherwise state "None"}

**What changed**: {Description}
**Why**: {Reason}
**How to migrate**: {Step-by-step guide}

---

## ⚙️ Configuration Changes

### New Settings

```json
{
  // New configuration options with inline comments
}
```

### Recommended Settings

**For Development**:

```json
{
  // Dev config
}
```

**For Production**:

```json
{
  // Prod config
}
```

---

## 📚 Documentation

- **Changelog**: See [CHANGELOG.md](CHANGELOG.md) for detailed changes
- **Upgrade Guide**: See upgrade section in [CHANGELOG.md](CHANGELOG.md#upgrading-to-{version})
- **Configuration**: See [Configuration Guide](docs/user-guide/CONFIGURATION_GUIDE.md)
- **API Reference**: See [API Endpoints](docs/api/ENDPOINTS.md)

---

## 🙏 Acknowledgments

{Thank contributors, testers, bug reporters, etc.}

---

## 📦 Installation

### New Installation

**Via Grafana CLI**:

```bash
grafana-cli plugins install jorgeancal-zagalin-app {VERSION}
```

**Via Docker**:

```bash
docker run -d \
  -p 3000:3000 \
  -e "GF_INSTALL_PLUGINS=jorgeancal-zagalin-app {VERSION}" \
  grafana/grafana:latest
```

**Via Helm**:

```yaml
plugins:
  - jorgeancal-zagalin-app {VERSION}
```

### Upgrade from Previous Version

```bash
grafana-cli plugins upgrade jorgeancal-zagalin-app
# Restart Grafana
```

**Docker Compose**:

```bash
docker compose pull
docker compose up -d
```

---

## 🔍 What's Next

{Sneak peek at what's coming in the next release}

- Feature preview 1
- Feature preview 2
- Feature preview 3

---

## 🐛 Known Issues

{IF ANY - otherwise state "None known"}

- Issue 1: Description and workaround
- Issue 2: Description and workaround

---

## 📞 Support

- **Documentation**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app#readme
- **Issues**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues
- **Discussions**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/discussions

---

## 🎉 Try It Out!

1. Install/upgrade Zagalin
2. Navigate to a Grafana dashboard
3. Click the floating chat button
4. Try out the new features!

**Example queries to try**:

- "Show me the error rate for my services"
- "Explain this panel to me"
- "Find traces with high latency"

---

**Full Changelog**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/blob/main/CHANGELOG.md
