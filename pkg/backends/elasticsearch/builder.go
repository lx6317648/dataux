package elasticsearch

import (
	"fmt"
	"strings"

	u "github.com/araddon/gou"
	"github.com/araddon/qlbridge/exec"
	"github.com/araddon/qlbridge/expr"
	"github.com/dataux/dataux/pkg/models"
	"github.com/dataux/dataux/vendor/mixer/proxy"
)

var (
	_ = u.EMPTY

	// Ensure that we implement the Exec Visitor interface
	_ exec.Visitor = (*Builder)(nil)
	//_ exec.JobRunner = (*Builder)(nil)
)

// Create Job made up of sub-tasks in DAG that is the
//   plan for execution of this query/job
func BuildSqlJob(svr *models.ServerCtx, writer models.ResultWriter, schemaDb, sqlText string) (*exec.SqlJob, error) {

	stmt, err := expr.ParseSql(sqlText)
	if err != nil {
		return nil, err
	}

	builder := NewBuilder(svr, writer, schemaDb)
	ex, err := stmt.Accept(builder)

	if err != nil {
		return nil, err
	}
	if ex == nil {
		// If No Error, and no Exec Tasks, then we already wrote results
		return nil, nil
	}
	tasks, ok := ex.(exec.Tasks)
	if !ok {
		return nil, fmt.Errorf("expected tasks but got: %T", ex)
	}
	return &exec.SqlJob{tasks, stmt}, nil
}

// This is a Sql Plan Builder that chooses backends
//   and routes/manages Requests
type Builder struct {
	schema   *models.Schema
	writer   models.ResultWriter
	svr      *models.ServerCtx
	where    expr.Node
	distinct bool
	children exec.Tasks
}

func NewBuilder(svr *models.ServerCtx, writer models.ResultWriter, db string) *Builder {
	m := Builder{writer: writer, svr: svr}
	m.schema = svr.Schema(db)
	return &m
}

func (m *Builder) VisitSelect(stmt *expr.SqlSelect) (interface{}, error) {
	u.Debugf("VisitSelect %+v", stmt)

	if sysVar := stmt.SysVariable(); len(sysVar) > 0 {
		return m.VisitSysVariable(stmt)
	}

	tblName := ""
	if len(stmt.From) > 1 {
		return nil, fmt.Errorf("join not implemented")
	}
	tblName = strings.ToLower(stmt.From[0].Name)

	tbl, _ := m.schema.Table(tblName)
	if tbl == nil {
		u.Errorf("Could not find table for '%s'.'%s'", m.schema.Db, tblName)
		return nil, fmt.Errorf("Could not find '%v'.'%v' schema", m.schema.Db, tblName)
	}

	es := NewSqlToEs(tbl)
	u.Debugf("sqltoes: %#v", es)
	resp, err := es.Query(stmt)
	if err != nil {
		u.Error(err)
		return nil, err
	}

	//u.Debugf("found : %#v", resp)

	//panic("here")
	rw := NewMysqlResultWriter(stmt, resp, tbl)

	if err := rw.Finalize(); err != nil {
		u.Error(err)
		return nil, err
	}
	return nil, m.writer.WriteResult(rw.rs)

	tasks := make(exec.Tasks, 0)
	/*
		// Create our Datasource Reader
		var source datasource.DataSource
		if len(stmt.From) == 1 {
			from := stmt.From[0]
			if from.Name != "" && from.Source == nil {
				source = m.conf.DataSource(m.connInfo, from.Name)
				//u.Debugf("source: %T", source)
				in := NewSourceScanner(from.Name, source)
				tasks.Add(in)
			}

		} else {
			// if we have a join?
		}

		u.Debugf("has where? %v", stmt.Where != nil)
		if stmt.Where != nil {
			switch {
			case stmt.Where.Source != nil:
				u.Warnf("Found un-supported subquery: %#v", stmt.Where)
			case stmt.Where.Expr != nil:
				where := NewWhere(stmt.Where.Expr)
				tasks.Add(where)
			default:
				u.Warnf("Found un-supported where type: %#v", stmt.Where)
			}

		}

		// Add a Projection
		projection := NewProjection(stmt)
		tasks.Add(projection)
	*/

	return tasks, nil
}

func (m *Builder) VisitSysVariable(stmt *expr.SqlSelect) (interface{}, error) {
	u.Debugf("VisitSysVariable %+v", stmt)
	switch sysVar := stmt.SysVariable(); sysVar {
	case "@@max_allowed_packet":
		r, _ := proxy.BuildSimpleSelectResult(MaxAllowedPacket, []byte(sysVar), nil)
		return nil, m.writer.WriteResult(r)
	default:
		u.Errorf("unknown var: %v", sysVar)
		return nil, fmt.Errorf("Unrecognized System Variable: %v", sysVar)
	}
	return nil, exec.ErrNotImplemented
}

func (m *Builder) VisitInsert(stmt *expr.SqlInsert) (interface{}, error) {
	u.Debugf("VisitInsert %+v", stmt)
	return nil, exec.ErrNotImplemented
}

func (m *Builder) VisitUpsert(stmt *expr.SqlUpsert) (interface{}, error) {
	u.Debugf("VisitUpsert %+v", stmt)
	return nil, exec.ErrNotImplemented
}

func (m *Builder) VisitDelete(stmt *expr.SqlDelete) (interface{}, error) {
	u.Debugf("VisitDelete %+v", stmt)
	return nil, exec.ErrNotImplemented
}

func (m *Builder) VisitUpdate(stmt *expr.SqlUpdate) (interface{}, error) {
	u.Debugf("VisitUpdate %+v", stmt)
	return nil, exec.ErrNotImplemented
}

func (m *Builder) VisitShow(stmt *expr.SqlShow) (interface{}, error) {
	u.Debugf("VisitShow %+v", stmt)
	return nil, exec.ErrNotImplemented
}

func (m *Builder) VisitDescribe(stmt *expr.SqlDescribe) (interface{}, error) {
	u.Debugf("VisitDescribe %+v", stmt)
	return nil, exec.ErrNotImplemented
}

func (m *Builder) VisitPreparedStmt(stmt *expr.PreparedStatement) (interface{}, error) {
	u.Debugf("VisitPreparedStmt %+v", stmt)
	return nil, exec.ErrNotImplemented
}