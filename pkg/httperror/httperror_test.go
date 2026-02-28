package httperror_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/braginantonev/mhserver/pkg/httperror"
)

var errTest error = errors.New("hi stranger")

func TestCompareWith(t *testing.T) {
	cases := [...]struct {
		name          string
		base_error    httperror.HttpError
		target_error  httperror.HttpError
		expected_same bool
	}{
		{
			name:          "Good external http error",
			base_error:    httperror.NewExternalHttpError(errTest, http.StatusNotFound),
			target_error:  httperror.NewExternalHttpError(errTest, http.StatusNotFound),
			expected_same: true,
		},
		{
			name:          "Good internal http error",
			base_error:    httperror.NewInternalHttpError(errTest, "TestCompareWith"),
			target_error:  httperror.NewInternalHttpError(errTest, "TestCompareWith"),
			expected_same: true,
		},
		{
			name:         "Bad http codes",
			base_error:   httperror.NewExternalHttpError(errTest, http.StatusNotFound),
			target_error: httperror.NewExternalHttpError(errTest, http.StatusMethodNotAllowed),
		},
		{
			name:         "bad description",
			base_error:   httperror.NewExternalHttpError(errors.New("i'm bad"), http.StatusBadGateway),
			target_error: httperror.NewExternalHttpError(errTest, http.StatusBadGateway),
		},
		{
			name:         "bad types",
			base_error:   httperror.NewInternalHttpError(errTest, "some func"),
			target_error: httperror.NewExternalHttpError(errTest, http.StatusBadRequest),
		},
		{
			name:         "compare with nil target",
			base_error:   httperror.NewExternalHttpError(errTest, http.StatusBadRequest),
			target_error: nil,
		},
		{
			name:          "compare nils",
			base_error:    nil,
			target_error:  nil,
			expected_same: true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if res := errors.Is(test.base_error, test.target_error); res != test.expected_same {
				t.Errorf("expected same: %t, but got %t", test.expected_same, res)
				t.Logf("\tbase error: %v\n\ttarget error: %v", test.base_error, test.target_error)
			}
		})
	}
}

//Todo: test Write()
