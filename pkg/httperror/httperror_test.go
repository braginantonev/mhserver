package httperror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/braginantonev/mhserver/pkg/httperror"
)

var errTest error = errors.New("hi stranger")

func TestCompareWith(t *testing.T) {
	cases := []struct {
		name          string
		http_err      httperror.HttpError
		test_http_err httperror.HttpError
		expected_err  error
	}{
		{
			name:          "Good external http error",
			http_err:      httperror.NewExternalHttpError(errTest, http.StatusNotFound),
			test_http_err: httperror.NewExternalHttpError(errTest, http.StatusNotFound),
			expected_err:  nil,
		},
		{
			name:          "Good internal http error",
			http_err:      httperror.NewInternalHttpError(errTest, "TestCompareWith"),
			test_http_err: httperror.NewInternalHttpError(errTest, "TestCompareWith"),
			expected_err:  nil,
		},
		{
			name:          "Bad http codes",
			http_err:      httperror.NewExternalHttpError(errTest, http.StatusNotFound),
			test_http_err: httperror.NewExternalHttpError(errTest, http.StatusMethodNotAllowed),
			expected_err:  fmt.Errorf(httperror.BAD_CODE, http.StatusNotFound, http.StatusMethodNotAllowed),
		},
		{
			name:          "Bad errors",
			http_err:      httperror.NewInternalHttpError(errTest, ""),
			test_http_err: httperror.NewInternalHttpError(fmt.Errorf("t"), ""),
			expected_err:  fmt.Errorf(httperror.BAD_ERROR, errTest, "t"),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := test.http_err.CompareWith(test.test_http_err)
			if test.expected_err == nil && err != nil {
				t.Errorf("expected nil error, but got %s", err.Error())
				return
			}

			if err != nil && test.expected_err != nil && err.Error() != test.expected_err.Error() {
				t.Errorf("expected: \"%s\",\nbut got: \"%s\"", test.expected_err.Error(), err.Error())
			}
		})
	}
}

//Todo: test Write()
