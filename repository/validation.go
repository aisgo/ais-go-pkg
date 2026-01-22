package repository

import (
	"fmt"
	"regexp"
	"strings"
)

/* ========================================================================
 * SQL 安全校验器
 * ========================================================================
 * 职责: 防止 OrderBy/Select/Joins 注入风险
 * 设计: 白名单模式 + 黑名单防御
 * ======================================================================== */

var (
	// 列名白名单正则：仅允许字母、数字、下划线、点号（表别名）
	// 格式: column 或 table.column 或 table.column AS alias
	columnPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?(\s+AS\s+[a-zA-Z_][a-zA-Z0-9_]*)?$`)

	// 排序方向白名单
	orderDirections = map[string]bool{
		"ASC":  true,
		"DESC": true,
		"asc":  true,
		"desc": true,
	}

	// SQL 危险关键字黑名单
	dangerousKeywords = []string{
		"DROP", "DELETE", "UPDATE", "INSERT", "TRUNCATE", "ALTER", "CREATE",
		"GRANT", "REVOKE", "EXEC", "EXECUTE", "UNION", "INTO", "OUTFILE",
		"LOAD_FILE", "DUMPFILE", "--", "/*", "*/", ";", "SLEEP", "BENCHMARK",
	}
)

// ValidationError SQL 校验错误
type ValidationError struct {
	Field   string // OrderBy/Select/Joins
	Value   string
	Reason  string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("SQL validation failed for %s: %s (value: %s, reason: %s)",
		e.Field, e.Message, e.Value, e.Reason)
}

// ValidateOrderBy 校验排序字符串
//
// 允许格式:
//   - "column ASC"
//   - "column DESC"
//   - "table.column ASC"
//   - "col1 ASC, col2 DESC"
func ValidateOrderBy(orderBy string) error {
	if strings.TrimSpace(orderBy) == "" {
		return nil // 空字符串允许
	}

	// 检查危险关键字
	if err := checkDangerousKeywords(orderBy, "OrderBy"); err != nil {
		return err
	}

	// 解析多个排序字段（逗号分隔）
	parts := strings.Split(orderBy, ",")
	for _, part := range parts {
		if err := validateSingleOrderBy(strings.TrimSpace(part)); err != nil {
			return err
		}
	}

	return nil
}

// validateSingleOrderBy 校验单个排序字段
func validateSingleOrderBy(orderBy string) error {
	if orderBy == "" {
		return nil
	}

	// 分割为 "column" 和 "direction"
	fields := strings.Fields(orderBy)
	if len(fields) == 0 || len(fields) > 2 {
		return &ValidationError{
			Field:   "OrderBy",
			Value:   orderBy,
			Reason:  "invalid_format",
			Message: "must be 'column' or 'column ASC/DESC'",
		}
	}

	// 校验列名
	column := fields[0]
	if err := validateColumnName(column); err != nil {
		return &ValidationError{
			Field:   "OrderBy",
			Value:   orderBy,
			Reason:  "invalid_column",
			Message: err.Error(),
		}
	}

	// 校验排序方向（如果存在）
	if len(fields) == 2 {
		direction := fields[1]
		if !orderDirections[direction] {
			return &ValidationError{
				Field:   "OrderBy",
				Value:   orderBy,
				Reason:  "invalid_direction",
				Message: fmt.Sprintf("direction must be ASC or DESC, got: %s", direction),
			}
		}
	}

	return nil
}

// ValidateSelect 校验选择字段
//
// 允许格式:
//   - []string{"id", "name"}
//   - []string{"users.id", "users.name"}
//   - []string{"COUNT(*) AS count"} (聚合函数)
func ValidateSelect(selects []string) error {
	if len(selects) == 0 {
		return nil // 空数组允许
	}

	for _, sel := range selects {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}

		// 检查危险关键字
		if err := checkDangerousKeywords(sel, "Select"); err != nil {
			return err
		}

		// 允许聚合函数: COUNT(*), SUM(column), AVG(column) 等
		if isAggregateFunction(sel) {
			continue
		}

		// 校验普通列名
		if err := validateColumnName(sel); err != nil {
			return &ValidationError{
				Field:   "Select",
				Value:   sel,
				Reason:  "invalid_column",
				Message: err.Error(),
			}
		}
	}

	return nil
}

// ValidateJoins 校验连接查询
//
// 允许格式:
//   - "LEFT JOIN orders ON orders.user_id = users.id"
//   - "INNER JOIN profiles ON profiles.user_id = users.id"
func ValidateJoins(joins []string) error {
	if len(joins) == 0 {
		return nil // 空数组允许
	}

	for _, join := range joins {
		join = strings.TrimSpace(join)
		if join == "" {
			continue
		}

		// 检查危险关键字
		if err := checkDangerousKeywords(join, "Joins"); err != nil {
			return err
		}

		// 校验 JOIN 语法
		if err := validateJoinSyntax(join); err != nil {
			return err
		}
	}

	return nil
}

// validateColumnName 校验列名
func validateColumnName(column string) error {
	col := strings.TrimSpace(column)
	if col == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	// 检查是否匹配白名单模式
	if !columnPattern.MatchString(col) {
		return fmt.Errorf("column name contains invalid characters: %s", col)
	}

	return nil
}

// validateJoinSyntax 校验 JOIN 语法
func validateJoinSyntax(join string) error {
	upperJoin := strings.ToUpper(join)

	// 必须包含 JOIN 关键字
	if !strings.Contains(upperJoin, "JOIN") {
		return &ValidationError{
			Field:   "Joins",
			Value:   join,
			Reason:  "missing_join_keyword",
			Message: "must contain JOIN keyword",
		}
	}

	// 必须包含 ON 条件
	if !strings.Contains(upperJoin, " ON ") {
		return &ValidationError{
			Field:   "Joins",
			Value:   join,
			Reason:  "missing_on_clause",
			Message: "must contain ON clause",
		}
	}

	// 允许的 JOIN 类型
	validJoinTypes := []string{"INNER JOIN", "LEFT JOIN", "RIGHT JOIN", "FULL JOIN", "CROSS JOIN", "JOIN"}
	hasValidType := false
	for _, jt := range validJoinTypes {
		if strings.Contains(upperJoin, jt) {
			hasValidType = true
			break
		}
	}

	if !hasValidType {
		return &ValidationError{
			Field:   "Joins",
			Value:   join,
			Reason:  "invalid_join_type",
			Message: "must use valid JOIN type (INNER/LEFT/RIGHT/FULL/CROSS)",
		}
	}

	return nil
}

// checkDangerousKeywords 检查危险关键字
func checkDangerousKeywords(value, field string) error {
	upperValue := strings.ToUpper(value)

	for _, keyword := range dangerousKeywords {
		// 使用单词边界匹配，避免误判 created_at 等合法列名
		// 例如：匹配 "DROP TABLE" 但不匹配 "created_at"
		if isKeywordMatch(upperValue, keyword) {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Reason:  "dangerous_keyword",
				Message: fmt.Sprintf("contains dangerous keyword: %s", keyword),
			}
		}
	}

	return nil
}

// isKeywordMatch 检查关键字是否匹配（使用单词边界）
func isKeywordMatch(text, keyword string) bool {
	// 特殊字符直接匹配
	if keyword == "--" || keyword == "/*" || keyword == "*/" || keyword == ";" {
		return strings.Contains(text, keyword)
	}

	// 单词关键字：检查前后是否为单词边界
	idx := strings.Index(text, keyword)
	if idx == -1 {
		return false
	}

	// 检查前面是否为单词边界
	if idx > 0 {
		prevChar := text[idx-1]
		if isWordChar(prevChar) {
			return false
		}
	}

	// 检查后面是否为单词边界
	endIdx := idx + len(keyword)
	if endIdx < len(text) {
		nextChar := text[endIdx]
		if isWordChar(nextChar) {
			return false
		}
	}

	return true
}

// isWordChar 检查字符是否为单词字符（字母、数字、下划线）
func isWordChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '_'
}

// isAggregateFunction 检查是否为聚合函数
func isAggregateFunction(sel string) bool {
	upperSel := strings.ToUpper(strings.TrimSpace(sel))

	aggregateFuncs := []string{"COUNT(", "SUM(", "AVG(", "MAX(", "MIN(", "GROUP_CONCAT("}
	for _, fn := range aggregateFuncs {
		if strings.HasPrefix(upperSel, fn) {
			return true
		}
	}

	return false
}
