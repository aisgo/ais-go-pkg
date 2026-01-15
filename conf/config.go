package conf

import (
	"bytes"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

/* ========================================================================
 * Config Loader - 配置加载器
 * ========================================================================
 * 职责: 统一配置加载，支持 YAML / JSON / 环境变量
 * 技术: Viper
 * ======================================================================== */

// Loader 定义配置加载接口
type Loader interface {
	Load(config any) error
}

type viperLoader struct {
	configPath string
	configName string
	configType string
	envPrefix  string
}

var envPlaceholderPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-(.*?))?\}`)

func expandEnvPlaceholders(raw string) string {
	return envPlaceholderPattern.ReplaceAllStringFunc(raw, func(match string) string {
		sub := envPlaceholderPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}

		name := sub[1]
		def := ""
		if len(sub) >= 3 {
			def = sub[2]
		}

		// 兼容 bash 的 ${VAR:-default} 语义：未设置或为空字符串时使用 default
		if val, ok := os.LookupEnv(name); ok && val != "" {
			return val
		}
		if def != "" {
			return def
		}
		return ""
	})
}

// NewLoader 创建一个新的配置加载器
// configPath: 配置文件目录
// configName: 配置文件名 (不含扩展名)
// configType: 配置文件类型 (yaml, json 等)
func NewLoader(configPath, configName, configType string) Loader {
	return &viperLoader{
		configPath: configPath,
		configName: configName,
		configType: configType,
		envPrefix:  "APP", // 默认环境变量前缀
	}
}

// NewLoaderWithEnvPrefix 创建带自定义环境变量前缀的配置加载器
func NewLoaderWithEnvPrefix(configPath, configName, configType, envPrefix string) Loader {
	return &viperLoader{
		configPath: configPath,
		configName: configName,
		configType: configType,
		envPrefix:  envPrefix,
	}
}

func (l *viperLoader) Load(config any) error {
	// 先让 viper 帮我们定位配置文件（支持 AddConfigPath + SetConfigName 的搜索逻辑）
	finder := viper.New()
	finder.AddConfigPath(l.configPath)
	finder.SetConfigName(l.configName)
	finder.SetConfigType(l.configType)

	finder.SetEnvPrefix(l.envPrefix)
	finder.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	finder.AutomaticEnv()

	if err := finder.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	configFile := finder.ConfigFileUsed()

	// 再读取一次配置：在进入 viper 解析前，做 ${VAR} / ${VAR:-default} 的环境变量占位符展开
	v := viper.New()
	v.SetEnvPrefix(l.envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configFile != "" {
		raw, err := os.ReadFile(configFile)
		if err != nil {
			return err
		}
		expanded := expandEnvPlaceholders(string(raw))

		v.SetConfigType(l.configType)
		if err := v.ReadConfig(bytes.NewBufferString(expanded)); err != nil {
			return err
		}
	}

	return v.Unmarshal(config)
}
