package skill

import (
	"context"
	"log/slog"
	"strings"

	"github.com/icooclaw/icooclaw/internal/storage"
)

type Skill struct {
	Name        string
	Description string
	Content     string
	AlwaysLoad  bool
	Enabled     bool
}

type Loader struct {
	storage    *storage.Storage
	logger     *slog.Logger
	loaded     []Skill
	alwaysLoad []Skill
}

func NewLoader(storage *storage.Storage, logger *slog.Logger) *Loader {
	return &Loader{
		storage:    storage,
		logger:     logger,
		loaded:     make([]Skill, 0),
		alwaysLoad: make([]Skill, 0),
	}
}

func (l *Loader) Load(ctx context.Context) error {
	l.logger.Info("Loading skills")

	skills, err := l.storage.GetEnabledSkills()
	if err != nil {
		l.logger.Warn("Failed to load skills from database", "error", err)
	}

	l.loadBuiltInSkills()

	for _, skill := range skills {
		l.loaded = append(l.loaded, Skill{
			Name:        skill.Name,
			Description: skill.Description,
			Content:     skill.Content,
			AlwaysLoad:  skill.AlwaysLoad,
			Enabled:     skill.Enabled,
		})
		if skill.AlwaysLoad {
			l.alwaysLoad = append(l.alwaysLoad, Skill{
				Name:        skill.Name,
				Description: skill.Description,
				Content:     skill.Content,
				AlwaysLoad:  skill.AlwaysLoad,
				Enabled:     skill.Enabled,
			})
		}
	}

	l.logger.Info("Skills loaded", "count", len(l.loaded))
	return nil
}

func (l *Loader) loadBuiltInSkills() {
	builtInSkills := GetBuiltInSkills()

	for _, skill := range builtInSkills {
		l.loaded = append(l.loaded, skill)
		if skill.AlwaysLoad {
			l.alwaysLoad = append(l.alwaysLoad, skill)
		}
	}
}

func (l *Loader) GetLoaded() []Skill {
	return l.loaded
}

func (l *Loader) GetAlwaysLoad() []Skill {
	return l.alwaysLoad
}

func (l *Loader) GetByName(name string) *Skill {
	for _, skill := range l.loaded {
		if strings.EqualFold(skill.Name, name) {
			return &skill
		}
	}
	return nil
}

func (l *Loader) Reload(ctx context.Context) error {
	l.loaded = make([]Skill, 0)
	l.alwaysLoad = make([]Skill, 0)
	return l.Load(ctx)
}
