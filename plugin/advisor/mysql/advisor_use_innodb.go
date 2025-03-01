package mysql

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
)

const (
	innoDB              string = "innodb"
	defaultStorageEngin string = "default_storage_engine"
)

var (
	_ advisor.Advisor = (*UseInnoDBAdvisor)(nil)
	_ ast.Visitor     = (*useInnoDBChecker)(nil)
)

func init() {
	advisor.Register(advisor.MySQL, advisor.MySQLUseInnoDB, &UseInnoDBAdvisor{})
}

// UseInnoDBAdvisor is the advisor checking for using InnoDB engine.
type UseInnoDBAdvisor struct {
}

// Check checks for using InnoDB engine.
func (*UseInnoDBAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &useInnoDBChecker{
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

type useInnoDBChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
}

// Enter implements the ast.Visitor interface.
func (v *useInnoDBChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := advisor.Ok
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		for _, option := range node.Options {
			if option.Tp == ast.TableOptionEngine && strings.ToLower(option.StrValue) != innoDB {
				code = advisor.NotInnoDBEngine
				break
			}
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			// TABLE OPTION
			if spec.Tp == ast.AlterTableOption {
				for _, option := range spec.Options {
					if option.Tp == ast.TableOptionEngine && strings.ToLower(option.StrValue) != innoDB {
						code = advisor.NotInnoDBEngine
						break
					}
				}
			}
		}
	// SET
	case *ast.SetStmt:
		for _, variable := range node.Variables {
			if strings.ToLower(variable.Name) == defaultStorageEngin {
				// Return lowercase
				text, err := restoreNode(variable.Value, format.RestoreNameLowercase)
				if err != nil {
					v.adviceList = append(v.adviceList, advisor.Advice{
						Status:  v.level,
						Code:    advisor.Internal,
						Title:   "Internal error for use InnoDB rule",
						Content: fmt.Sprintf("\"%s\" meet internal error %q", in.Text(), err.Error()),
					})
					continue
				}
				if text != innoDB {
					code = advisor.NotInnoDBEngine
					break
				}
			}
		}
	}

	if code != advisor.Ok {
		v.adviceList = append(v.adviceList, advisor.Advice{
			Status:  v.level,
			Code:    code,
			Title:   v.title,
			Content: fmt.Sprintf("\"%s\" doesn't use InnoDB engine", in.Text()),
		})
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*useInnoDBChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
