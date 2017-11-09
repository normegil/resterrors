package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/normegil/resterrors"
	"github.com/normegil/resterrors/middleware"
	"github.com/sirupsen/logrus"
)

const TEXT_NO_ERROR = "NoError"

func TestErrorHandler(t *testing.T) {
	testcases := []struct {
		Name         string
		Error        error
		CodeToAssign int
		ErrorHandler middleware.ErrorHandlerFunc
		ExpectedBody string
	}{
		{
			Name:         "NoError",
			Error:        nil,
			ExpectedBody: "NoError",
		},
		{
			Name:  "Error:NoCode",
			Error: errors.New("TestErr"),
		},
		{
			Name:         "Error:WithCode",
			Error:        errors.New("TestErr"),
			CodeToAssign: 40001,
		},
		{
			Name:         "Error:NoCode;WithHandler",
			Error:        errors.New("TestErr"),
			ExpectedBody: "test",
			ErrorHandler: func(w http.ResponseWriter, err error) {
				w.Write([]byte("test"))
			},
		},
		{
			Name:         "Error:WithCode:WithHandler",
			Error:        errors.New("TestErr"),
			CodeToAssign: 40001,
			ExpectedBody: "40001",
			ErrorHandler: func(w http.ResponseWriter, err error) {
				e := err.(resterrors.ErrWithCode)
				w.Write([]byte(strconv.Itoa(e.Code())))
			},
		},
	}
	for _, testdata := range testcases {
		t.Run(testdata.Name, func(t *testing.T) {
			factory := middleware.ErrorHandlerFactory{
				Logger:                logrus.New(),
				ErrorHandlerFunc:      testdata.ErrorHandler,
				ErrorCodeAssignerFunc: CodeAssignerForTest(testdata.CodeToAssign).Handle,
			}

			result := httptest.NewRecorder()
			factory.New(EndpointForTest{testdata.Error}.Handle).Handle(result, httptest.NewRequest("GET", "http://localhost/", strings.NewReader("")), nil)

			expected := getExpectedBody(testdata.ExpectedBody, testdata.Error, testdata.CodeToAssign)
			if expected != string(result.Body.Bytes()) {
				t.Errorf("Response (%s) doesn't correspond to expected output (%s)", string(result.Body.Bytes()), testdata.ExpectedBody)
			}
		})
	}
}

type CodeAssignerForTest int

func (c CodeAssignerForTest) Handle(err error) error {
	if 0 != c {
		return resterrors.NewErrWithCode(int(c), err)
	}
	return err
}

type EndpointForTest struct {
	Error error
}

func (e EndpointForTest) Handle(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) error {
	if nil != e.Error {
		return e.Error
	}
	w.Write([]byte(TEXT_NO_ERROR))
	return nil
}

func getExpectedBody(expectedBody string, err error, codeToAssign int) string {
	expected := expectedBody
	if "" == expectedBody {
		errToHandle := err
		if 0 != codeToAssign {
			errToHandle = resterrors.NewErrWithCode(codeToAssign, err)
		}
		expectedResponse := httptest.NewRecorder()
		resterrors.Handler{logrus.New()}.Handle(expectedResponse, errToHandle)
		expected = string(expectedResponse.Body.Bytes())
	}
	return expected
}
