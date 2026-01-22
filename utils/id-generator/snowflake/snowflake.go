package snowflake

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/bwmarrin/snowflake"
)

/* ========================================================================
 * Snowflake ID Generator - 雪花算法 ID 生成器
 * ========================================================================
 * 职责: 生成分布式唯一 ID
 * 特点:
 *   - 64 位整数
 *   - 趋势递增
 *   - 不依赖中心化服务
 * ID 结构:
 *   - 1 位符号位 (始终为 0)
 *   - 41 位时间戳 (毫秒级，可用 69 年)
 *   - 10 位机器 ID (支持 1024 节点)
 *   - 12 位序列号 (每毫秒 4096 个 ID)
 *
 * 环境变量配置:
 *   SNOWFLAKE_NODE_ID: 设置节点 ID (0-1023)
 * ======================================================================== */

const (
	// MaxNodeID 最大节点 ID (10 位)
	MaxNodeID = 1023
	// DefaultNodeID 默认节点 ID (从环境变量读取，默认为 0)
	DefaultNodeID = 0
	// EnvNodeID 环境变量名
	EnvNodeID = "SNOWFLAKE_NODE_ID"
)

var (
	globalNode *snowflake.Node
	once       sync.Once
)

// Generator ID 生成器
type Generator struct {
	node *snowflake.Node
}

// NewGenerator 创建新的 ID 生成器
// nodeID: 节点 ID (0-1023)
//
// 如果在同一进程需要多个独立的生成器，可以使用此方法。
// 否则建议直接使用全局函数 Generate()。
func NewGenerator(nodeID int64) (*Generator, error) {
	if nodeID < 0 || nodeID > MaxNodeID {
		return nil, &ConfigError{
			Field:   "nodeID",
			Value:   nodeID,
			Message: "nodeID must be between 0 and 1023",
		}
	}

	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		return nil, err
	}

	return &Generator{node: node}, nil
}

// MustNewGenerator 创建新的 ID 生成器，失败时 panic
func MustNewGenerator(nodeID int64) *Generator {
	gen, err := NewGenerator(nodeID)
	if err != nil {
		panic(err)
	}
	return gen
}

// Generate 生成雪花 ID
func (g *Generator) Generate() int64 {
	return g.node.Generate().Int64()
}

// GenerateString 生成雪花 ID（字符串格式）
func (g *Generator) GenerateString() string {
	return g.node.Generate().String()
}

// ========================================================================
// 全局函数（使用默认节点 ID）
// ========================================================================

// initNode 初始化全局雪花节点（仅执行一次）
// 节点 ID 从环境变量 SNOWFLAKE_NODE_ID 读取，默认为 0
func initNode() error {
	nodeID, err := getEnvNodeID()
	if err != nil {
		return err
	}

	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		return &ConfigError{
			Field:   "nodeID",
			Value:   nodeID,
			Message: err.Error(),
		}
	}

	globalNode = node
	return nil
}

// Generate 生成雪花 ID
// 使用环境变量 SNOWFLAKE_NODE_ID 指定的节点 ID（默认为 0）
//
// 注意: 在多实例部署环境中，必须为每个实例配置不同的节点 ID！
func Generate() int64 {
	once.Do(func() {
		if err := initNode(); err != nil {
			panic(err.Error())
		}
	})

	return globalNode.Generate().Int64()
}

// GenerateString 生成雪花 ID（字符串格式）
func GenerateString() string {
	return snowflake.ID(Generate()).String()
}

// Parse 解析雪花 ID
// 返回: 时间戳（毫秒）、节点 ID
func Parse(id int64) (timestamp int64, nodeID int64) {
	sid := snowflake.ID(id)
	return sid.Time(), sid.Node()
}

// ========================================================================
// 辅助函数
// ========================================================================

// getEnvNodeID 从环境变量获取节点 ID
func getEnvNodeID() (int64, error) {
	val := os.Getenv(EnvNodeID)
	if val == "" {
		return DefaultNodeID, nil
	}

	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: invalid integer", EnvNodeID, val)
	}

	if id < 0 || id > MaxNodeID {
		return 0, &ConfigError{
			Field:   EnvNodeID,
			Value:   id,
			Message: "nodeID must be between 0 and 1023",
		}
	}

	return id, nil
}

// ConfigError 配置错误
type ConfigError struct {
	Field   string
	Value   int64
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + "=" + strconv.FormatInt(e.Value, 10) + ": " + e.Message
}
