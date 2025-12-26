# Publishing to Grafana Catalog

This guide walks through the process of publishing Zagalin to the official Grafana Plugin Catalog.

## Overview

Publishing to the Grafana Plugin Catalog allows users to discover and install your plugin directly from Grafana's UI. The process involves:

1. Preparing your plugin
2. Signing the plugin
3. Submitting for review
4. Addressing feedback
5. Plugin approval and publication

## Prerequisites

### Before You Submit

- [ ] Plugin is fully functional and tested
- [ ] All acceptance criteria met
- [ ] E2E tests passing
- [ ] Documentation complete
- [ ] README.md with clear description
- [ ] CHANGELOG.md with version history
- [ ] LICENSE file (Apache 2.0)
- [ ] Screenshots and logo prepared
- [ ] plugin.json metadata complete

### Required Information

1. **Grafana.com Account**: Sign up at https://grafana.com/
2. **GitHub Repository**: Public repo with source code
3. **Plugin ID**: Must match `plugin.json` (e.g., `jorgeancal-zagalin-app`)
4. **Signing Key**: For code signing (see [Plugin Signing](./signing.md))

## Step 1: Prepare Plugin Metadata

### plugin.json Requirements

Ensure your `plugin.json` contains all required fields:

```json
{
  "type": "app",
  "name": "Zagalin",
  "id": "jorgeancal-zagalin-app",
  "info": {
    "description": "AI-powered assistant for Grafana with LLM integration",
    "author": {
      "name": "Jorge Andreu Calatayud",
      "url": "https://github.com/jorgeancal"
    },
    "keywords": [
      "ai",
      "assistant",
      "llm",
      "gpt",
      "chat",
      "query-generation"
    ],
    "logos": {
      "small": "img/logo.svg",
      "large": "img/logo.svg"
    },
    "links": [
      {
        "name": "GitHub",
        "url": "https://github.com/jorgesnotebook/jorgeancal-zagalin-app"
      },
      {
        "name": "License",
        "url": "https://github.com/jorgesnotebook/jorgeancal-zagalin-app/blob/main/LICENSE"
      }
    ],
    "screenshots": [
      {
        "name": "Floating Chat Interface",
        "path": "img/screenshots/floating-chat.png"
      },
      {
        "name": "Dashboard Context Awareness",
        "path": "img/screenshots/dashboard-context.png"
      },
      {
        "name": "Configuration",
        "path": "img/screenshots/configuration.png"
      }
    ],
    "version": "1.0.0",
    "updated": "2025-12-26"
  },
  "dependencies": {
    "grafanaDependency": ">=10.0.0",
    "plugins": [
      {
        "type": "app",
        "name": "LLM App",
        "id": "grafana-llm-app",
        "version": ">=0.1.0"
      }
    ]
  }
}
```

### Key Fields Explained

**Required**:
- `type`: Plugin type (app, datasource, panel)
- `name`: Display name
- `id`: Unique identifier (must match folder name)
- `info.description`: Clear, concise description
- `info.author.name`: Your name or organization
- `info.version`: Semantic version (MAJOR.MINOR.PATCH)
- `dependencies.grafanaDependency`: Minimum Grafana version

**Recommended**:
- `info.keywords`: For search optimization
- `info.logos`: Small (40×40) and large (200×200) logos
- `info.links`: GitHub, docs, support links
- `info.screenshots`: Feature demonstrations
- `dependencies.plugins`: Required plugins

## Step 2: Create Quality Documentation

### README.md

Your README should include:

```markdown
# Zagalin - AI Assistant for Grafana

Clear one-line description

## Features
- Feature 1
- Feature 2
- Feature 3

## Installation
Step-by-step installation instructions

## Configuration
How to configure the plugin

## Usage
How to use the plugin with examples

## Screenshots
![Screenshot](img/screenshot.png)

## Requirements
- Grafana version
- Dependencies

## Support
- GitHub Issues link
- Community forum link

## License
Apache 2.0
```

### CHANGELOG.md

Maintain a clear changelog following [Keep a Changelog](https://keepachangelog.com/):

```markdown
# Changelog

## [1.0.0] - 2025-12-26

### Added
- Initial release
- Context-aware chat interface
- Conversation history management
- Query generation support
- Troubleshooting assistance

### Changed
- N/A (initial release)

### Fixed
- N/A (initial release)

### Security
- N/A (initial release)
```

## Step 3: Sign Your Plugin

Plugin signing is **required** for public distribution. See [Plugin Signing Guide](./signing.md) for detailed instructions.

### Quick Sign Process

1. **Create signing key** (if first time)
2. **Set environment variables**
3. **Sign the plugin**
4. **Verify signature**

```bash
# Sign plugin
npm run sign

# Verify MANIFEST.txt exists
ls -la dist/MANIFEST.txt
```

## Step 4: Build for Production

### Create Distribution Build

```bash
# Clean previous builds
rm -rf dist/

# Install dependencies
npm ci

# Run tests
npm test
npm run e2e

# Type check
npm run typecheck

# Lint
npm run lint

# Build
npm run build

# Build backend (if applicable)
mage -v

# Sign
npm run sign
```

### Verify Build Output

Check that `dist/` contains:

```
dist/
├── MANIFEST.txt          # Signature file
├── module.js             # Frontend bundle
├── module.js.map         # Source map
├── plugin.json           # Metadata
├── README.md             # Documentation
├── CHANGELOG.md          # Version history
├── LICENSE               # License file
├── img/                  # Assets
│   ├── logo.svg
│   └── screenshots/
└── gpx_zagalin_*         # Backend binaries (if applicable)
```

## Step 5: Package Plugin

### Create Release Archive

```bash
# Create zip archive
cd dist
zip -r ../jorgeancal-zagalin-app-1.0.0.zip .
cd ..

# Verify archive
unzip -l jorgeancal-zagalin-app-1.0.0.zip
```

### GitHub Release

1. **Create Git Tag**:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. **Create GitHub Release**:
   - Go to https://github.com/jorgesnotebook/jorgeancal-zagalin-app/releases
   - Click "Draft a new release"
   - Select tag `v1.0.0`
   - Title: "v1.0.0 - Initial Release"
   - Description: Copy from CHANGELOG.md
   - Attach `jorgeancal-zagalin-app-1.0.0.zip`
   - Publish release

## Step 6: Submit to Grafana

### Submission Process

1. **Navigate to**: https://grafana.com/auth/sign-in/
2. **Sign in** with your Grafana.com account
3. **Go to**: "My Plugins" → "Submit a Plugin"
4. **Fill out submission form**:

#### Basic Information
- **Plugin ID**: `jorgeancal-zagalin-app`
- **Plugin Name**: Zagalin
- **Plugin Type**: App
- **Short Description**: AI-powered assistant for Grafana
- **Long Description**: (Detailed description from README)

#### Repository Information
- **GitHub URL**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app
- **Release URL**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/releases/tag/v1.0.0
- **Download URL**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/releases/download/v1.0.0/jorgeancal-zagalin-app-1.0.0.zip

#### Media Assets
- **Logo**: Upload `img/logo.svg`
- **Screenshots**: Upload from `img/screenshots/`
- **Video** (optional): Demo video URL

#### Technical Details
- **Grafana Version**: 10.0.0+
- **Dependencies**: grafana-llm-app
- **Signing**: Signed (include MANIFEST.txt)

#### Testing Information
- **Test Environment**: Provide Docker setup or test instance
- **Test Account**: (If applicable)
- **Test Instructions**: (How to test the plugin)

### What the Review Team Checks

- [ ] Plugin loads without errors
- [ ] All features work as described
- [ ] No security vulnerabilities
- [ ] Code quality meets standards
- [ ] Documentation is clear and complete
- [ ] Plugin is properly signed
- [ ] Screenshots match actual functionality
- [ ] Dependencies are declared correctly
- [ ] License is compatible (Apache 2.0)
- [ ] No trademark violations

## Step 7: Review Process

### Timeline

- **Submission**: Instant
- **Initial Review**: 1-2 weeks
- **Feedback**: Variable
- **Approval**: After all issues addressed

### Communication

- **Email**: Updates sent to your Grafana.com account email
- **GitHub**: Reviewers may open issues on your repo
- **Forum**: May be discussed on community forum

### Common Feedback

1. **Documentation Issues**
   - Missing installation steps
   - Unclear configuration
   - No screenshots

2. **Technical Issues**
   - Plugin doesn't load
   - Errors in console
   - Performance problems

3. **Security Concerns**
   - Unsafe API calls
   - XSS vulnerabilities
   - Unvalidated inputs

4. **Metadata Problems**
   - Incorrect plugin.json
   - Missing dependencies
   - Wrong version format

## Step 8: Address Feedback

### Iterating on Feedback

1. **Acknowledge**: Respond to reviewer comments
2. **Fix Issues**: Make required changes
3. **Test**: Verify fixes work
4. **Update**: Push new version
5. **Resubmit**: Notify reviewers

### Version Bumps

Follow semantic versioning:

```bash
# Bug fix: 1.0.0 → 1.0.1
npm version patch

# New feature: 1.0.1 → 1.1.0
npm version minor

# Breaking change: 1.1.0 → 2.0.0
npm version major
```

## Step 9: Approval and Publication

### Once Approved

- Plugin appears in Grafana Plugin Catalog
- Users can install via Grafana CLI
- Available in Grafana Cloud
- Listed on https://grafana.com/grafana/plugins/

### Post-Publication

1. **Announce**: Blog post, social media, forum
2. **Monitor**: Watch for issues, questions
3. **Support**: Respond to user feedback
4. **Update**: Release bug fixes and new features

## Maintaining Your Plugin

### Regular Updates

- **Bug Fixes**: Release promptly
- **Security Patches**: Critical priority
- **New Features**: Based on user feedback
- **Dependency Updates**: Keep current with Grafana

### Versioning Strategy

```
1.0.0 - Initial release
1.0.1 - Bug fix
1.1.0 - New feature (backward compatible)
1.2.0 - Another feature
2.0.0 - Breaking change
```

### Update Process

1. Make changes
2. Update CHANGELOG.md
3. Bump version in plugin.json
4. Run tests
5. Build and sign
6. Create GitHub release
7. Submit update to Grafana

## Best Practices

### Before Submission

- [ ] Test on multiple Grafana versions
- [ ] Test on different browsers
- [ ] Get feedback from beta users
- [ ] Review all documentation
- [ ] Check for typos and broken links
- [ ] Verify all screenshots are current
- [ ] Test installation process

### During Review

- [ ] Respond promptly to feedback
- [ ] Be open to suggestions
- [ ] Test reviewer's concerns
- [ ] Document any limitations
- [ ] Provide test environment if needed

### After Approval

- [ ] Monitor GitHub issues
- [ ] Engage with community
- [ ] Keep documentation updated
- [ ] Release updates regularly
- [ ] Maintain backward compatibility

## Troubleshooting Submission

### Plugin Not Loading in Review

**Possible Causes**:
- Missing dependencies in plugin.json
- Unsigned plugin
- Incorrect plugin ID
- Build errors

**Solution**: Test installation on clean Grafana instance

### Signature Verification Failed

**Possible Causes**:
- Expired signing key
- Modified files after signing
- MANIFEST.txt missing

**Solution**: Re-sign plugin with valid key

### Review Taking Too Long

**Steps**:
1. Check email for feedback
2. Check GitHub issues
3. Ping on community forum
4. Be patient (team is small)

## Getting Help

### Resources

- **Grafana Plugin Tools**: https://grafana.com/developers/plugin-tools/
- **Community Forum**: https://community.grafana.com/
- **GitHub Discussions**: Plugin-specific Q&A
- **Stack Overflow**: Tag with `grafana` and `grafana-plugin`

### Support Channels

1. **Documentation**: Check plugin tools docs first
2. **Community Forum**: Ask general questions
3. **GitHub Issues**: Report bugs, request features
4. **Email**: Direct support for submission issues

## Related Documentation

- [Build Process](./build.md)
- [Plugin Signing](./signing.md)
- [Docker Deployment](./docker.md)
- [Contributing Guidelines](../contributing/guidelines.md)
