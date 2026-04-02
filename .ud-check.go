package main

import (
  "fmt"

  uniquedialect "github.com/fatballfish/uniquedialect"
  internalparser "github.com/fatballfish/uniquedialect/internal/parser"
  tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
)

func main() {
  parsed, err := internalparser.ParseOne("ALTER TABLE `users` DROP FOREIGN KEY IF EXISTS `fk_users_org_id`", uniquedialect.DialectMySQL)
  fmt.Printf("err=%v\n", err)
  fmt.Printf("kind=%v status=%v type=%T\n", parsed.Kind, parsed.Status, parsed.NativeAST)
  if stmt, ok := parsed.NativeAST.(*tidbast.AlterTableStmt); ok {
    fmt.Printf("specs=%d\n", len(stmt.Specs))
    for i, s := range stmt.Specs {
      fmt.Printf("spec[%d]: tp=%v ifExists=%v name=%q\n", i, s.Tp, s.IfExists, s.Name)
    }
  }
}
