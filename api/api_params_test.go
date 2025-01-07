package api

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
)

func TestPathParamToID(t *testing.T) {
	req, err := http.NewRequest("GET", "/path/{id}", nil)
	if err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"id": "7e0e6e24a6e29faf11d022dca77826fe8b8a000aff5ea27e16650d03acefc93c",
	}
	req = mux.SetURLVars(req, vars)

	id, err := PathParamToID(req, "id")
	if err != nil {
		t.Errorf("PathParamToID returned error: %v", err)
	}

	expectedID := [32]uint8{
		0x7e, 0xe, 0x6e, 0x24, 0xa6, 0xe2, 0x9f, 0xaf,
		0x11, 0xd0, 0x22, 0xdc, 0xa7, 0x78, 0x26, 0xfe,
		0x8b, 0x8a, 0x0, 0xa, 0xff, 0x5e, 0xa2, 0x7e,
		0x16, 0x65, 0xd, 0x3, 0xac, 0xef, 0xc9, 0x3c,
	}

	if id != expectedID {
		t.Errorf("PathParamToID returned unexpected ID: %v", id)
	}
}

func TestPathParamToID_Invalid(t *testing.T) {
	tests := []struct {
		name string
		id   string
		err  string
	}{
		{
			name: "empty id",
			id:   "",
			err:  "invalid_params: Invalid parameter",
		},
		{
			name: "wrong format",
			id:   "0",
			err:  "invalid_params: Invalid parameter",
		},
		{
			name: "wrong length",
			id:   "67",
			err:  "invalid_params: Invalid parameter",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/path/{id}", nil)
			if err != nil {
				t.Fatal(err)
			}

			vars := map[string]string{
				"id": test.id,
			}
			req = mux.SetURLVars(req, vars)

			_, err = PathParamToID(req, "id")
			if err.Error() != test.err {
				t.Errorf("wrong error, expected: %v, got: %v", err, test.err)
			}
		})
	}
}
