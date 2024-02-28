package bizerr

import (
	"fmt"
	"io"

	"github.com/bentoml/yatai/common/utils/errorx"
)

type (
	BizLogicError interface {
		HttpStatusCode() int
		Message() string
		ErrorNo() int
		error
	}
	bizLogicError struct {
		httpCode int
		errorNo  int
		err      error
	}
)

/* errors usages:
errors.New("msg with stack")
errors.WithStack(err)
errors.Wrap(err, "more message with stack")
errors.WithMessage(err, "more messsage without more stack")
*/

// New create new bizlogic error with either string or error or errors.Wrap
func New(httpCode int, newErr interface{}, bizcode ...int) BizLogicError {
	err := errorx.EnsureStack(newErr, 2)
	he := bizLogicError{
		httpCode: httpCode,
		err:      err,
	}
	if len(bizcode) == 1 {
		he.errorNo = bizcode[0]
	}
	return he
}

func (be bizLogicError) HttpStatusCode() int {
	return be.httpCode
}

func (be bizLogicError) ErrorNo() int {
	return be.errorNo
}

func (be bizLogicError) Message() string {
	return be.Error()
}

func (be bizLogicError) Error() string {
	return be.err.Error()
}

func (be bizLogicError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "%+v", be.err)
			return
		}
		fallthrough
	case 's', 'q':
		_, _ = io.WriteString(s, be.err.Error())
	}
}
