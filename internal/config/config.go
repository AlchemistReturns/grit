package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type QuestionsConfig struct {
	Pool   []string `mapstructure:"pool"`
	Window int      `mapstructure:"window"`
}

type ThresholdsConfig struct {
	Complexity      float64 `mapstructure:"complexity"`
	AIReflectLines  int     `mapstructure:"ai_reflect_lines"`
	DeadTimeMinutes int     `mapstructure:"dead_time_minutes"`
	UndoSpikeLines  int     `mapstructure:"undo_spike_lines"`
	PasteLines      int     `mapstructure:"paste_lines"`
}

type WatchConfig struct {
	Extensions    []string            `mapstructure:"extensions"`
	LanguageNames map[string][]string `mapstructure:"language_names"`
}

type ExportConfig struct {
	Path       string `mapstructure:"path"`
	AutoExport bool   `mapstructure:"auto_export"`
}

type DeepReflectConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	OutputDir string `mapstructure:"output_dir"`
}

type Config struct {
	// Legacy fields kept for backward compatibility
	WatchExtensions     []string `mapstructure:"watch_extensions"`
	ComplexityThreshold float64  `mapstructure:"complexity_threshold"`

	Questions   QuestionsConfig   `mapstructure:"questions"`
	Thresholds  ThresholdsConfig  `mapstructure:"thresholds"`
	Watch       WatchConfig       `mapstructure:"watch"`
	Export      ExportConfig      `mapstructure:"export"`
	DeepReflect DeepReflectConfig `mapstructure:"deep_reflect"`
}

var defaultPool = []string{
	"What's the hardest part of this change?",
	"What would you do differently next time?",
	"What assumption are you making that could be wrong?",
	"Did you have to look anything up? What was the gap?",
	"Is this change reversible? If not, why not?",
	"Who else needs to know about this change?",
	"What did you cut from the original plan, and was that the right call?",
	"What could break that you haven't tested?",
	"What's the simplest thing you could have done here?",
	"What would you tell a new engineer about this code tomorrow?",
	"What slowed you down the most on this change?",
	"What's the riskiest part of this commit?",
}

var defaults = Config{
	WatchExtensions:     []string{".go", ".js", ".ts", ".py", ".rs", ".java", ".c", ".cpp"},
	ComplexityThreshold: 10.0,
	Questions: QuestionsConfig{
		Pool:   defaultPool,
		Window: 5,
	},
	Thresholds: ThresholdsConfig{
		Complexity:      10.0,
		AIReflectLines:  15,
		DeadTimeMinutes: 40,
		UndoSpikeLines:  20,
		PasteLines:      30,
	},
	Watch: WatchConfig{
		Extensions: []string{".go", ".js", ".ts", ".py", ".rs", ".java", ".c", ".cpp"},
	},
	Export: ExportConfig{
		Path:       "~/.grit/exports",
		AutoExport: false,
	},
	DeepReflect: DeepReflectConfig{
		Enabled:   true,
		OutputDir: "~/.grit/reflections",
	},
}

func GritDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".grit")
}

func DBPath() string {
	return filepath.Join(GritDir(), "store.db")
}

func EnsureGritDir() error {
	return os.MkdirAll(GritDir(), 0755)
}

func Load() (*Config, error) {
	viper.SetDefault("watch_extensions", defaults.WatchExtensions)
	viper.SetDefault("complexity_threshold", defaults.ComplexityThreshold)
	viper.SetDefault("questions.pool", defaults.Questions.Pool)
	viper.SetDefault("questions.window", defaults.Questions.Window)
	viper.SetDefault("thresholds.complexity", defaults.Thresholds.Complexity)
	viper.SetDefault("thresholds.ai_reflect_lines", defaults.Thresholds.AIReflectLines)
	viper.SetDefault("thresholds.dead_time_minutes", defaults.Thresholds.DeadTimeMinutes)
	viper.SetDefault("thresholds.undo_spike_lines", defaults.Thresholds.UndoSpikeLines)
	viper.SetDefault("thresholds.paste_lines", defaults.Thresholds.PasteLines)
	viper.SetDefault("watch.extensions", defaults.Watch.Extensions)
	viper.SetDefault("export.path", defaults.Export.Path)
	viper.SetDefault("export.auto_export", defaults.Export.AutoExport)
	viper.SetDefault("deep_reflect.enabled", defaults.DeepReflect.Enabled)
	viper.SetDefault("deep_reflect.output_dir", defaults.DeepReflect.OutputDir)

	viper.SetConfigName(".grit")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	_ = viper.ReadInConfig()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return &defaults, nil
	}

	// Migrate legacy fields to new structure
	if len(cfg.Watch.Extensions) == 0 {
		if len(cfg.WatchExtensions) > 0 {
			cfg.Watch.Extensions = cfg.WatchExtensions
		} else {
			cfg.Watch.Extensions = defaults.Watch.Extensions
		}
	}
	if cfg.Thresholds.Complexity == 0 {
		if cfg.ComplexityThreshold > 0 {
			cfg.Thresholds.Complexity = cfg.ComplexityThreshold
		} else {
			cfg.Thresholds.Complexity = defaults.Thresholds.Complexity
		}
	}
	if len(cfg.Questions.Pool) == 0 {
		cfg.Questions.Pool = defaults.Questions.Pool
	}
	if cfg.Questions.Window == 0 {
		cfg.Questions.Window = defaults.Questions.Window
	}
	if cfg.Thresholds.AIReflectLines == 0 {
		cfg.Thresholds.AIReflectLines = defaults.Thresholds.AIReflectLines
	}
	if cfg.Thresholds.DeadTimeMinutes == 0 {
		cfg.Thresholds.DeadTimeMinutes = defaults.Thresholds.DeadTimeMinutes
	}
	if cfg.Thresholds.UndoSpikeLines == 0 {
		cfg.Thresholds.UndoSpikeLines = defaults.Thresholds.UndoSpikeLines
	}
	if cfg.Thresholds.PasteLines == 0 {
		cfg.Thresholds.PasteLines = defaults.Thresholds.PasteLines
	}

	return &cfg, nil
}

// ExpandPath expands ~ to the home directory in a path string.
func ExpandPath(p string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
