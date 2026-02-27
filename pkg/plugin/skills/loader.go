package skills

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed *.md
var skillFiles embed.FS

// SkillMetadata contains parsed frontmatter from skill markdown files
type SkillMetadata struct {
	Name          string              `yaml:"name"`
	Description   string              `yaml:"description"`
	Triggers      map[string][]string `yaml:"triggers"`
	MinConfidence int                 `yaml:"min_confidence"`
	MaxSteps      int                 `yaml:"max_steps"`
	ScoreWeights  map[string]int      `yaml:"score_weights"`

	// Optional fields
	RequiresPanelContext     bool     `yaml:"requires_panel_context"`
	RequiresDashboardContext bool     `yaml:"requires_dashboard_context"`
	NegativeSignals          []string `yaml:"negative_signals"`
	SupportsTemplate         bool     `yaml:"supports_template"`
	TemplatePlaceholder      string   `yaml:"template_placeholder"`
}

// Skill represents a loaded skill with metadata and content
type Skill struct {
	Metadata SkillMetadata
	Content  string // The markdown content after frontmatter
	FileName string
}

// SkillRegistry manages loaded skills
type SkillRegistry struct {
	mu       sync.RWMutex
	skills   map[string]*Skill
	metadata map[string]*SkillMetadata
	loaded   bool
}

// NewSkillRegistry creates a new skill registry
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills:   make(map[string]*Skill),
		metadata: make(map[string]*SkillMetadata),
	}
}

// Load scans the embedded skills directory and loads all skill files
func (r *SkillRegistry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries, err := skillFiles.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, err := skillFiles.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read skill file %s: %w", entry.Name(), err)
		}

		skill, err := parseSkillFile(string(content), entry.Name())
		if err != nil {
			return fmt.Errorf("failed to parse skill file %s: %w", entry.Name(), err)
		}

		r.skills[skill.Metadata.Name] = skill
		r.metadata[skill.Metadata.Name] = &skill.Metadata
	}

	r.loaded = true
	return nil
}

// parseSkillFile parses a skill markdown file with YAML frontmatter
func parseSkillFile(content, fileName string) (*Skill, error) {
	// Check for frontmatter delimiter
	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("skill file must start with YAML frontmatter (---)")
	}

	// Find the closing frontmatter delimiter
	endIdx := strings.Index(content[4:], "\n---")
	if endIdx == -1 {
		return nil, fmt.Errorf("skill file missing closing frontmatter delimiter (---)")
	}

	frontmatter := content[4 : 4+endIdx]
	markdownContent := strings.TrimSpace(content[4+endIdx+4:])

	var metadata SkillMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Set defaults
	if metadata.MinConfidence == 0 {
		metadata.MinConfidence = 40
	}
	if metadata.MaxSteps == 0 {
		metadata.MaxSteps = 5
	}

	return &Skill{
		Metadata: metadata,
		Content:  markdownContent,
		FileName: fileName,
	}, nil
}

// GetMetadata returns the metadata for a skill without loading the full content
func (r *SkillRegistry) GetMetadata(name string) *SkillMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if meta, ok := r.metadata[name]; ok {
		return meta
	}
	return nil
}

// GetContent returns the full content for a skill
// In development mode, this reloads the file each time to pick up changes
func (r *SkillRegistry) GetContent(name string) string {
	r.mu.RLock()
	skill, ok := r.skills[name]
	r.mu.RUnlock()

	if !ok {
		return ""
	}

	// For embedded files, just return the cached content
	// In a future enhancement, we could support file-based reloading for development
	return skill.Content
}

// GetSkill returns the full skill with metadata and content
func (r *SkillRegistry) GetSkill(name string) *Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.skills[name]
}

// ListSkills returns all available skill names
func (r *SkillRegistry) ListSkills() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	return names
}

// AllMetadata returns all skill metadata for scoring
func (r *SkillRegistry) AllMetadata() map[string]*SkillMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	result := make(map[string]*SkillMetadata, len(r.metadata))
	for k, v := range r.metadata {
		result[k] = v
	}
	return result
}

// ScoreInput scores input text against all skills and returns the best match
func (r *SkillRegistry) ScoreInput(input string, hasAlertSource, hasDashboardContext, hasPanelContext bool) (bestSkill string, bestScore int) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inputLower := strings.ToLower(input)

	for name, meta := range r.metadata {
		score := r.scoreSkill(inputLower, meta, hasAlertSource, hasDashboardContext, hasPanelContext)
		if score > bestScore {
			bestScore = score
			bestSkill = name
		}
	}

	return bestSkill, bestScore
}

// scoreSkill scores input against a single skill's triggers
func (r *SkillRegistry) scoreSkill(inputLower string, meta *SkillMetadata, hasAlertSource, hasDashboardContext, hasPanelContext bool) int {
	// Check context requirements
	if meta.RequiresDashboardContext && !hasDashboardContext {
		return 0
	}
	if meta.RequiresPanelContext && !hasPanelContext {
		return 0
	}

	score := 0

	// Special case: alert_source gives maximum score for incident_investigate
	if hasAlertSource && meta.Name == "incident_investigate" {
		if weight, ok := meta.ScoreWeights["alert_source"]; ok {
			return weight // Return immediately with highest score
		}
	}

	// Check negative signals first - reduce score if present
	for _, signal := range meta.NegativeSignals {
		if strings.Contains(inputLower, strings.ToLower(signal)) {
			score -= 50
		}
	}

	// Score each trigger category
	for category, triggers := range meta.Triggers {
		weight := 20 // default weight
		if w, ok := meta.ScoreWeights[category]; ok {
			weight = w
		}

		for _, trigger := range triggers {
			if strings.Contains(inputLower, strings.ToLower(trigger)) {
				score += weight
			}
		}
	}

	return score
}

// IsLoaded returns whether the registry has been loaded
func (r *SkillRegistry) IsLoaded() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.loaded
}
