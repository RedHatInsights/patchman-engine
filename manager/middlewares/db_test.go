package middlewares

import (
	"app/base/database"
	"app/base/utils"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func ginWithMethod(method string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, "/", nil)
	return c
}

func TestUseReadReplica(t *testing.T) {
	origEnabled := utils.CoreCfg.DBReadReplicaEnabled
	origHost := utils.CoreCfg.DBReadReplicaHost
	origPort := utils.CoreCfg.DBReadReplicaPort
	t.Cleanup(func() {
		utils.CoreCfg.DBReadReplicaEnabled = origEnabled
		utils.CoreCfg.DBReadReplicaHost = origHost
		utils.CoreCfg.DBReadReplicaPort = origPort
	})

	utils.CoreCfg.DBReadReplicaHost = "replica.example"
	utils.CoreCfg.DBReadReplicaPort = 5432

	utils.CoreCfg.DBReadReplicaEnabled = false
	assert.False(t, useReadReplica(ginWithMethod(http.MethodGet)))

	utils.CoreCfg.DBReadReplicaEnabled = true
	assert.True(t, useReadReplica(ginWithMethod(http.MethodGet)))
	assert.False(t, useReadReplica(ginWithMethod(http.MethodPost)))
	assert.False(t, useReadReplica(ginWithMethod(http.MethodPut)))
	assert.False(t, useReadReplica(ginWithMethod(http.MethodPatch)))
	assert.False(t, useReadReplica(ginWithMethod(http.MethodDelete)))

	utils.CoreCfg.DBReadReplicaHost = ""
	assert.False(t, useReadReplica(ginWithMethod(http.MethodGet)))
}

func TestDBWriterUsesRequestContext(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := ginWithMethod(http.MethodPost)
	c.Request = c.Request.WithContext(ctx)

	db := DBWriter(c)
	require.NotNil(t, db)
	assert.Equal(t, ctx, db.Statement.Context)

	writerPool := poolOf(t, db)
	primaryPool := poolOf(t, database.DB(ctx))
	assert.Same(t, primaryPool, writerPool)
}

func TestDBGetWithoutReplicaUsesPrimaryAndRequestContext(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	origEnabled := utils.CoreCfg.DBReadReplicaEnabled
	t.Cleanup(func() { utils.CoreCfg.DBReadReplicaEnabled = origEnabled })
	utils.CoreCfg.DBReadReplicaEnabled = false

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := ginWithMethod(http.MethodGet)
	c.Request = c.Request.WithContext(ctx)

	db := DB(c)
	require.NotNil(t, db)
	assert.Equal(t, ctx, db.Statement.Context)
	assert.Same(t, poolOf(t, database.DB(ctx)), poolOf(t, db))
}

func poolOf(t *testing.T, db *gorm.DB) any {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	return sqlDB
}

func TestHandlersDoNotCallDatabaseDBDirectly(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	roots := []string{
		filepath.Join(root, "manager", "controllers"),
		filepath.Join(root, "turnpike", "controllers"),
		filepath.Join(root, "base", "core", "probes.go"),
	}
	var offenders []string
	for _, start := range roots {
		err := filepath.WalkDir(start, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			offenders = append(offenders, directDatabaseDBCalls(t, path)...)
			return nil
		})
		require.NoError(t, err)
	}
	assert.Empty(t, offenders)
}

func directDatabaseDBCalls(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)
	dbName := importName(f, "app/base/database")
	baseName := importName(f, "app/base")
	if dbName == "" {
		return nil
	}
	var hits []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "DB" && sel.Sel.Name != "DBReadReplica") {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != dbName {
			return true
		}
		if sel.Sel.Name == "DB" && len(call.Args) == 1 && isBaseContext(call.Args[0], baseName) {
			return true
		}
		pos := fset.Position(call.Pos())
		hits = append(hits, pos.String())
		return true
	})
	return hits
}

func importName(f *ast.File, path string) string {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != path {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return filepath.Base(path)
	}
	return ""
}

func isBaseContext(arg ast.Expr, baseName string) bool {
	sel, ok := arg.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Context" || baseName == "" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == baseName
}
