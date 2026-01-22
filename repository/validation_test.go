package repository

import (
	"testing"
)

/* ========================================================================
 * ValidateOrderBy 测试
 * ======================================================================== */

func TestValidateOrderBy(t *testing.T) {
	tests := []struct {
		name    string
		orderBy string
		wantErr bool
	}{
		// 合法用例
		{"empty string", "", false},
		{"simple column ASC", "created_at ASC", false},
		{"simple column DESC", "id DESC", false},
		{"column without direction", "name", false},
		{"table.column", "users.name ASC", false},
		{"multiple fields", "status ASC, created_at DESC", false},
		{"lowercase direction", "id asc", false},

		// 注入攻击
		{"SQL injection - comment", "id--", true},
		{"SQL injection - union", "id UNION SELECT", true},
		{"SQL injection - drop", "id; DROP TABLE users", true},
		{"SQL injection - semicolon", "id;", true},
		{"SQL injection - sleep", "id, SLEEP(5)", true},
		{"invalid direction", "id RANDOM", true},
		{"too many parts", "id ASC DESC", true},
		{"special characters", "id@name", true},
		{"parenthesis", "COUNT(*)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrderBy(tt.orderBy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOrderBy(%q) error = %v, wantErr %v", tt.orderBy, err, tt.wantErr)
			}
		})
	}
}

/* ========================================================================
 * ValidateSelect 测试
 * ======================================================================== */

func TestValidateSelect(t *testing.T) {
	tests := []struct {
		name    string
		selects []string
		wantErr bool
	}{
		// 合法用例
		{"empty array", []string{}, false},
		{"single column", []string{"id"}, false},
		{"multiple columns", []string{"id", "name", "email"}, false},
		{"table.column", []string{"users.id", "users.name"}, false},
		{"aggregate function", []string{"COUNT(*) AS count"}, false},
		{"sum function", []string{"SUM(amount) AS total"}, false},

		// 注入攻击
		{"SQL injection - drop", []string{"id", "name; DROP TABLE users"}, true},
		{"SQL injection - union", []string{"* FROM users--"}, true},
		{"SQL injection - comment", []string{"id--"}, true},
		{"SQL injection - semicolon", []string{"id;"}, true},
		{"special characters", []string{"id@name"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSelect(tt.selects)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSelect(%v) error = %v, wantErr %v", tt.selects, err, tt.wantErr)
			}
		})
	}
}

/* ========================================================================
 * ValidateJoins 测试
 * ======================================================================== */

func TestValidateJoins(t *testing.T) {
	tests := []struct {
		name    string
		joins   []string
		wantErr bool
	}{
		// 合法用例
		{"empty array", []string{}, false},
		{"inner join", []string{"INNER JOIN orders ON orders.user_id = users.id"}, false},
		{"left join", []string{"LEFT JOIN profiles ON profiles.user_id = users.id"}, false},
		{"right join", []string{"RIGHT JOIN departments ON departments.id = users.dept_id"}, false},
		{"multiple joins", []string{
			"LEFT JOIN orders ON orders.user_id = users.id",
			"INNER JOIN products ON products.id = orders.product_id",
		}, false},

		// 非法用例
		{"missing JOIN keyword", []string{"orders ON orders.user_id = users.id"}, true},
		{"missing ON clause", []string{"LEFT JOIN orders"}, true},
		{"SQL injection - drop", []string{"LEFT JOIN orders ON 1=1; DROP TABLE users--"}, true},
		{"SQL injection - union", []string{"LEFT JOIN orders ON 1=1 UNION SELECT"}, true},
		{"SQL injection - comment", []string{"LEFT JOIN orders-- ON orders.user_id = users.id"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJoins(tt.joins)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJoins(%v) error = %v, wantErr %v", tt.joins, err, tt.wantErr)
			}
		})
	}
}

/* ========================================================================
 * validateColumnName 测试
 * ======================================================================== */

func TestValidateColumnName(t *testing.T) {
	tests := []struct {
		name    string
		column  string
		wantErr bool
	}{
		{"simple column", "user_id", false},
		{"table.column", "users.id", false},
		{"snake_case", "created_at", false},
		{"with alias", "users.name AS user_name", false},

		{"empty", "", true},
		{"with space", "user id", true},
		{"special char", "user@id", true},
		{"sql keyword", "DROP TABLE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateColumnName(tt.column)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateColumnName(%q) error = %v, wantErr %v", tt.column, err, tt.wantErr)
			}
		})
	}
}

/* ========================================================================
 * WithOrderBy/WithSelect/WithJoins 集成测试
 * ======================================================================== */

func TestWithOrderBy_Integration(t *testing.T) {
	t.Run("valid orderBy is applied", func(t *testing.T) {
		opt := &QueryOption{}
		WithOrderBy("created_at DESC")(opt)

		if opt.OrderBy != "created_at DESC" {
			t.Errorf("expected OrderBy to be set, got empty")
		}
	})

	t.Run("invalid orderBy is rejected", func(t *testing.T) {
		opt := &QueryOption{}
		WithOrderBy("id; DROP TABLE users")(opt)

		if opt.OrderBy != "" {
			t.Errorf("expected OrderBy to be empty (rejected), got %q", opt.OrderBy)
		}
	})
}

func TestWithSelect_Integration(t *testing.T) {
	t.Run("valid select is applied", func(t *testing.T) {
		opt := &QueryOption{}
		WithSelect("id", "name")(opt)

		if len(opt.Select) != 2 {
			t.Errorf("expected 2 select fields, got %d", len(opt.Select))
		}
	})

	t.Run("invalid select is rejected", func(t *testing.T) {
		opt := &QueryOption{}
		WithSelect("id", "name; DROP TABLE users")(opt)

		if len(opt.Select) != 0 {
			t.Errorf("expected Select to be empty (rejected), got %v", opt.Select)
		}
	})
}

func TestWithJoins_Integration(t *testing.T) {
	t.Run("valid join is applied", func(t *testing.T) {
		opt := &QueryOption{}
		WithJoins("LEFT JOIN orders ON orders.user_id = users.id")(opt)

		if len(opt.Joins) != 1 {
			t.Errorf("expected 1 join, got %d", len(opt.Joins))
		}
	})

	t.Run("invalid join is rejected", func(t *testing.T) {
		opt := &QueryOption{}
		WithJoins("LEFT JOIN orders; DROP TABLE users")(opt)

		if len(opt.Joins) != 0 {
			t.Errorf("expected Joins to be empty (rejected), got %v", opt.Joins)
		}
	})
}
