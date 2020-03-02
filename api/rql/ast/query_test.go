package ast

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/stretchr/testify/suite"
)

type QueryTestSuite struct {
	asttest.Suite
}

// QTC => QueryTestCase
func (s *QueryTestSuite) QTC(rawQuery interface{}, trueV interface{}) {
	q := Query()
	if s.NoError(q.Unmarshal(rawQuery)) {
		switch t := trueV.(type) {
		case rql.Entry:
			s.True(q.(rql.EntryPredicate).EvalEntry(t))
		case *rql.EntrySchema:
			s.True(q.(rql.EntrySchemaPredicate).EvalEntrySchema(t))
		default:
			s.FailNow("t is not an Entry/EntrySchema value, it is instead %T", trueV)
		}
	}
}

func (s *QueryTestSuite) TestCanUnmarshalAllThePrimariesAndTheirExpressions() {
	// These are in the same order as they're created in the primary
	// directory
	s.testPrimaryWithNPEAction("action", func(action string) interface{} {
		e := rql.Entry{}
		e.Actions = []string{action}
		return e
	})

	s.testPrimaryWithNPETime("atime", func(t time.Time) interface{} {
		e := rql.Entry{}
		e.Attributes.SetAtime(t)
		return e
	})

	s.testPrimaryWithNPEString("cname", func(s string) interface{} {
		e := rql.Entry{}
		e.CName = s
		return e
	})

	s.testPrimaryWithNPETime("crtime", func(t time.Time) interface{} {
		e := rql.Entry{}
		e.Attributes.SetCrtime(t)
		return e
	})

	s.testPrimaryWithNPETime("ctime", func(t time.Time) interface{} {
		e := rql.Entry{}
		e.Attributes.SetCtime(t)
		return e
	})

	s.testPrimaryWithNPEString("kind", func(s string) interface{} {
		es := &rql.EntrySchema{}
		es.SetPath(s)
		return es
	})

	s.testPrimaryWithNPETime("mtime", func(t time.Time) interface{} {
		e := rql.Entry{}
		e.Attributes.SetMtime(t)
		return e
	})

	s.testPrimaryWithNPEString("name", func(s string) interface{} {
		e := rql.Entry{}
		e.Name = s
		return e
	})

	s.testPrimaryWithNPEString("path", func(s string) interface{} {
		e := rql.Entry{}
		e.Path = s
		return e
	})

	s.testPrimaryWithNPENumeric("size", func(n float64) interface{} {
		e := rql.Entry{}
		e.Attributes.SetSize(uint64(n))
		return e
	})

	s.testPrimaryWithPEObject("meta", func(metadata map[string]interface{}) interface{} {
		e := rql.Entry{}
		e.Metadata = metadata
		return e
	})
}

func (s *QueryTestSuite) TestCanUnmarshalPEPrimary() {
	e := rql.Entry{}
	e.Name = "foo"
	e.CName = "foo"
	s.QTC(s.A("AND", s.A("name", s.A("glob", "foo")), s.A("cname", s.A("glob", "foo"))), e)
	s.QTC(s.A("OR", s.A("name", s.A("glob", "bar")), s.A("cname", s.A("glob", "foo"))), e)
}

func (s *QueryTestSuite) TestUnmarshalErrors() {
	q := Query()
	s.UMETC(q, s.A(), "expected.*PE.*Primary", true)
	s.UMETC(q, s.A("name", 1), "expected.*NPE.*StringPredicate", false)
	s.UMETC(q, s.A("NOT", 1), "expected.*PE.*Primary", true)
	s.UMETC(q, s.A("AND", 1, 2), "expected.*PE.*Primary", false)
	s.UMETC(q, s.A("OR", 1, 2), "expected.*PE.*Primary", false)
}

func (s *QueryTestSuite) testPrimaryWithNPEAction(primaryName string, constructV func(string) interface{}) {
	s.QTC(s.A(primaryName, "exec"), constructV("exec"))
	s.QTC(s.A(primaryName, s.A("NOT", "exec")), constructV("list"))
	s.QTC(s.A(primaryName, s.A("AND", "exec", "exec")), constructV("exec"))
	s.QTC(s.A(primaryName, s.A("OR", "stream", "exec")), constructV("stream"))
}

func (s *QueryTestSuite) testPrimaryWithNPETime(primaryName string, constructV func(time.Time) interface{}) {
	s.QTC(s.A(primaryName, s.A(">", float64(500))), constructV(s.TM(1000)))
	s.QTC(s.A(primaryName, s.A("NOT", s.A(">", float64(500)))), constructV(s.TM(500)))
	s.QTC(s.A(primaryName, s.A("AND", s.A(">=", float64(500)), s.A("=", float64(500)))), constructV(s.TM(500)))
	s.QTC(s.A(primaryName, s.A("OR", s.A(">", float64(500)), s.A("=", float64(500)))), constructV(s.TM(500)))
}

func (s *QueryTestSuite) testPrimaryWithNPENumeric(primaryName string, constructV func(float64) interface{}) {
	s.QTC(s.A(primaryName, s.A(">", float64(500))), constructV(1000))
	s.QTC(s.A(primaryName, s.A("NOT", s.A(">", float64(500)))), constructV(500))
	s.QTC(s.A(primaryName, s.A("AND", s.A(">=", float64(500)), s.A("=", float64(500)))), constructV(500))
	s.QTC(s.A(primaryName, s.A("OR", s.A(">", float64(500)), s.A("=", float64(500)))), constructV(500))
}

func (s *QueryTestSuite) testPrimaryWithNPEString(primaryName string, constructV func(string) interface{}) {
	// Test that it can marshal all the atoms
	s.QTC(s.A(primaryName, s.A("glob", "foo")), constructV("foo"))
	s.QTC(s.A(primaryName, s.A("regex", "foo")), constructV("foo"))
	s.QTC(s.A(primaryName, s.A("=", "foo")), constructV("foo"))
	// Now test that it can marshal the operators
	s.QTC(s.A(primaryName, s.A("NOT", s.A("glob", "foo"))), constructV("bar"))
	s.QTC(s.A(primaryName, s.A("AND", s.A("glob", "*o*"), s.A("glob", "foo"))), constructV("foo"))
	s.QTC(s.A(primaryName, s.A("OR", s.A("glob", "foo"), s.A("glob", "bar"))), constructV("bar"))
}

func (s *QueryTestSuite) testPrimaryWithPEObject(primaryName string, constructV func(map[string]interface{}) interface{}) {
	// This helper saves some typing
	objAtom := func(val bool) interface{} {
		return s.A("object", s.A(s.A("key", "foo"), s.A("boolean", val)))
	}
	s.QTC(s.A(primaryName, objAtom(true)), constructV(map[string]interface{}{"foo": true}))
	s.QTC(s.A(primaryName, s.A("AND", objAtom(true), objAtom(true))), constructV(map[string]interface{}{"foo": true}))
	s.QTC(s.A(primaryName, s.A("OR", objAtom(false), objAtom(true))), constructV(map[string]interface{}{"foo": true}))
}

func TestQuery(t *testing.T) {
	suite.Run(t, new(QueryTestSuite))
}
