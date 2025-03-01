package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
	_ ast.Visitor     = (*tableNoFKChecker)(nil)
)

func init() {
	advisor.Register(advisor.MySQL, advisor.MySQLTableNoFK, &TableNoFKAdvisor{})
	advisor.Register(advisor.TiDB, advisor.MySQLTableNoFK, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &tableNoFKChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return checker.adviceList, nil
}

type tableNoFKChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
}

// Enter implements the ast.Visitor interface.
func (checker *tableNoFKChecker) Enter(in ast.Node) (ast.Node, bool) {
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, constraint := range node.Constraints {
			if constraint.Tp == ast.ConstraintForeignKey {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.TableHasFK,
					Title:   checker.title,
					Content: fmt.Sprintf("Foreign key is not allowed in the table `%s`", node.Table.Name),
				})
			}
		}
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			if spec.Tp == ast.AlterTableAddConstraint && spec.Constraint.Tp == ast.ConstraintForeignKey {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.TableHasFK,
					Title:   checker.title,
					Content: fmt.Sprintf("Foreign key is not allowed in the table `%s`", node.Table.Name),
				})
			}
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*tableNoFKChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
