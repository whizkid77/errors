package errors

import (
	"fmt"
	"io"
	"runtime"
	"testing"
)

var initpc, _, _, _ = runtime.Caller(0)

func TestFrameLine(t *testing.T) {
	var tests = []struct {
		Frame
		want int
	}{{
		Frame(initpc),
		10,
	}, {
		func() Frame {
			var pc, _, _, _ = runtime.Caller(0)
			return Frame(pc)
		}(),
		21,
	}, {
		func() Frame {
			var pc, _, _, _ = runtime.Caller(1)
			return Frame(pc)
		}(),
		29,
	}, {
		Frame(0), // invalid PC
		0,
	}}

	for _, tt := range tests {
		got := tt.Frame.line()
		want := tt.want
		if want != got {
			t.Errorf("Frame(%v): want: %v, got: %v", uintptr(tt.Frame), want, got)
		}
	}
}

type X struct{}

func (x X) val() Frame {
	var pc, _, _, _ = runtime.Caller(0)
	return Frame(pc)
}

func (x *X) ptr() Frame {
	var pc, _, _, _ = runtime.Caller(0)
	return Frame(pc)
}

func TestFrameFormat(t *testing.T) {
	var tests = []struct {
		Frame
		format string
		want   string
	}{{
		Frame(initpc),
		"%s",
		"stack_test.go",
	}, {
		Frame(initpc),
		"%+s",
		"github.com/whizkid77/errors.init\n" +
			"\t.+/github.com/whizkid77/errors/stack_test.go",
	}, {
		Frame(0),
		"%s",
		"unknown",
	}, {
		Frame(0),
		"%+s",
		"unknown",
	}, {
		Frame(initpc),
		"%d",
		"10",
	}, {
		Frame(0),
		"%d",
		"0",
	}, {
		Frame(initpc),
		"%n",
		"init",
	}, {
		func() Frame {
			var x X
			return x.ptr()
		}(),
		"%n",
		`\(\*X\).ptr`,
	}, {
		func() Frame {
			var x X
			return x.val()
		}(),
		"%n",
		"X.val",
	}, {
		Frame(0),
		"%n",
		"",
	}, {
		Frame(initpc),
		"%v",
		"stack_test.go:10",
	}, {
		Frame(initpc),
		"%+v",
		"github.com/whizkid77/errors.init\n" +
			"\t.+/github.com/whizkid77/errors/stack_test.go:10",
	}, {
		Frame(0),
		"%v",
		"unknown:0",
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.Frame, tt.format, tt.want)
	}
}

func TestFuncname(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{"", ""},
		{"runtime.main", "main"},
		{"github.com/whizkid77/errors.funcname", "funcname"},
		{"funcname", "funcname"},
		{"io.copyBuffer", "copyBuffer"},
		{"main.(*R).Write", "(*R).Write"},
	}

	for _, tt := range tests {
		got := funcname(tt.name)
		want := tt.want
		if got != want {
			t.Errorf("funcname(%q): want: %q, got %q", tt.name, want, got)
		}
	}
}

func TestStackTrace(t *testing.T) {
	tests := []struct {
		err  error
		want []string
	}{{
		New("ooh"), []string{
			"github.com/whizkid77/errors.TestStackTrace\n" +
				"\t.+/github.com/whizkid77/errors/stack_test.go:155",
		},
	}, {
		Annotate(New("ooh"), "ahh"), []string{
			"github.com/whizkid77/errors.TestStackTrace\n" +
				"\t.+/github.com/whizkid77/errors/stack_test.go:160", // this is the stack of Wrap, not New
		},
	}, {
		Cause(Annotate(New("ooh"), "ahh")), []string{
			"github.com/whizkid77/errors.TestStackTrace\n" +
				"\t.+/github.com/whizkid77/errors/stack_test.go:165", // this is the stack of New
		},
	}, {
		func() error { return New("ooh") }(), []string{
			`github.com/whizkid77/errors.(func·009|TestStackTrace.func1)` +
				"\n\t.+/github.com/whizkid77/errors/stack_test.go:170", // this is the stack of New
			"github.com/whizkid77/errors.TestStackTrace\n" +
				"\t.+/github.com/whizkid77/errors/stack_test.go:170", // this is the stack of New's caller
		},
	}, {
		Cause(func() error {
			return func() error {
				return Errorf("hello %s", fmt.Sprintf("world"))
			}()
		}()), []string{
			`github.com/whizkid77/errors.(func·010|TestStackTrace.func2.1)` +
				"\n\t.+/github.com/whizkid77/errors/stack_test.go:179", // this is the stack of Errorf
			`github.com/whizkid77/errors.(func·011|TestStackTrace.func2)` +
				"\n\t.+/github.com/whizkid77/errors/stack_test.go:180", // this is the stack of Errorf's caller
			"github.com/whizkid77/errors.TestStackTrace\n" +
				"\t.+/github.com/whizkid77/errors/stack_test.go:181", // this is the stack of Errorf's caller's caller
		},
	}}
	for i, tt := range tests {
		ste, ok := tt.err.(interface {
			StackTrace() StackTrace
		})
		if !ok {
			ste = tt.err.(interface {
				Cause() error
			}).Cause().(interface {
				StackTrace() StackTrace
			})
		}
		st := ste.StackTrace()
		for j, want := range tt.want {
			testFormatRegexp(t, i, st[j], "%+v", want)
		}
	}
}

// This comment helps to maintain original line numbers
// Perhaps this test is too fragile :)
func stackTrace() StackTrace {
	return NewStack(0).StackTrace()
	// This comment helps to maintain original line numbers
	// Perhaps this test is too fragile :)
}

func TestStackTraceFormat(t *testing.T) {
	tests := []struct {
		StackTrace
		format string
		want   string
	}{{
		nil,
		"%s",
		`\[\]`,
	}, {
		nil,
		"%v",
		`\[\]`,
	}, {
		nil,
		"%+v",
		"",
	}, {
		nil,
		"%#v",
		`\[\]errors.Frame\(nil\)`,
	}, {
		make(StackTrace, 0),
		"%s",
		`\[\]`,
	}, {
		make(StackTrace, 0),
		"%v",
		`\[\]`,
	}, {
		make(StackTrace, 0),
		"%+v",
		"",
	}, {
		make(StackTrace, 0),
		"%#v",
		`\[\]errors.Frame{}`,
	}, {
		stackTrace()[:2],
		"%s",
		`\[stack_test.go stack_test.go\]`,
	}, {
		stackTrace()[:2],
		"%v",
		`[stack_test.go:207 stack_test.go:254]`,
	}, {
		stackTrace()[:2],
		"%+v",
		"\n" +
			"github.com/whizkid77/errors.stackTrace\n" +
			"\t.+/github.com/whizkid77/errors/stack_test.go:211\n" +
			"github.com/whizkid77/errors.TestStackTraceFormat\n" +
			"\t.+/github.com/whizkid77/errors/stack_test.go:262",
	}, {
		stackTrace()[:2],
		"%#v",
		`\[\]errors.Frame{stack_test.go:211, stack_test.go:270}`,
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.StackTrace, tt.format, tt.want)
	}
}

func TestNewStack(t *testing.T) {
	got := NewStack(1).StackTrace()
	want := NewStack(1).StackTrace()
	if got[0] != want[0] {
		t.Errorf("NewStack(remove NewStack): want: %v, got: %v", want, got)
	}
	gotFirst := fmt.Sprintf("%+v", got[0])[0:15]
	if gotFirst != "testing.tRunner" {
		t.Errorf("NewStack(): want: %v, got: %+v", "testing.tRunner", gotFirst)
	}
}

func TestNewNoStackError(t *testing.T) {
	err := NewNoStackError("test error")
	err = Trace(err)
	err = Trace(err)
	result := fmt.Sprintf("%+v", err)
	if result != "test error" {
		t.Errorf("NewNoStackError(): want %s, got %v", "test error", result)
	}
}

func TestSuspendError(t *testing.T) {
	err := io.EOF
	err = SuspendStack(err)
	err = Trace(err)
	result := fmt.Sprintf("%+v", err)
	if result != "EOF" {
		t.Errorf("NewNoStackError(): want %s, got %v", "EOF", result)
	}
	if io.EOF != Cause(err) {
		t.Errorf("SuspendStackError can not got back origion error.")
	}
}

func TestSuspendTracedWithMessageError(t *testing.T) {
	tracedErr := Trace(io.EOF)
	tracedErr = WithStack(tracedErr)
	tracedErr = WithMessage(tracedErr, "1")
	tracedErr = SuspendStack(tracedErr)
	tracedErr = Trace(tracedErr)
	result := fmt.Sprintf("%+v", tracedErr)
	if result != "EOF\n1" {
		t.Errorf("NewNoStackError(): want %s, got %v", "EOF\n1", result)
	}
}
