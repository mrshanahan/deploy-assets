package manifest

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func pass[T any](x T) error {
	return nil
}

func noError(x error) error {
	if x == nil {
		return nil
	}
	return fmt.Errorf("expected nil error, but got %v", x)
}

func hasError(x error) error {
	if x != nil {
		return nil
	}
	return fmt.Errorf("expected non-nil error, but got nil")
}

func isNil(x any) error {
	v := reflect.ValueOf(x)
	if !v.IsValid() || v.IsNil() {
		return nil
	}
	return fmt.Errorf("expected nil, but got %v", x)
}

func isNotNil(x any) error {
	v := reflect.ValueOf(x)
	if v.IsValid() && !v.IsNil() {
		return nil
	}
	return fmt.Errorf("expected non-nil, but got nil")
}

func containsText(msg string) func(any) error {
	return func(x any) error {
		e, ok := x.(error)
		if !ok {
			return fmt.Errorf("META: expected error passed to validation function, instead got %v", x)
		}
		if !strings.Contains(e.Error(), msg) {
			return fmt.Errorf("expected error to contain '%s', but did not: %v", msg, e)
		}
		return nil
	}
}

func not(f func(any) error) func(any) error {
	return func(x any) error {
		if err := f(x); err != nil {
			return nil
		}
		return fmt.Errorf("expected error but got none")
	}
}

func and(fs ...func(any) error) func(any) error {
	return func(x any) error {
		for _, f := range fs {
			if err := f(x); err != nil {
				return err
			}
		}
		return nil
	}
}

func TestParseManifest(t *testing.T) {
	var tests = []struct {
		json         string
		manifestFunc func(any) error
		errFunc      func(any) error
	}{
		{"{}",
			isNil,
			and(
				containsText("required top-level key 'locations'"),
				containsText("required top-level key 'transport'"),
				containsText("required top-level key 'assets'"))},
		{"{\"locations\": []}",
			isNil,
			and(
				not(containsText("required top-level key 'locations'")),
				containsText("required top-level key 'transport'"),
				containsText("required top-level key 'assets'"))},
		{"{\"transport\": {}}",
			isNil,
			and(
				containsText("required top-level key 'locations'"),
				not(containsText("required top-level key 'transport'")),
				containsText("required top-level key 'assets'"))},
		{"{\"assets\": []}",
			isNil, and(
				containsText("required top-level key 'locations'"),
				containsText("required top-level key 'transport'"),
				not(containsText("required top-level key 'assets'")))},
		{"{\"assets\": {}}",
			isNil,
			containsText("top-level entry should be array of objects")},
		{"{\"locations\": [], \"transport\": {\"type\": \"foo\"}, \"assets\": []}",
			isNil,
			containsText("transport: unrecognized type 'foo'")},
		{"{\"locations\": [{ \"type\": \"local\", \"name\": \"local\" }, { \"type\": \"ssh\", \"name\": \"remote\", \"server\": \"foo.com:22\", \"username\": \"test\", \"key_file\": \"/home/test/.ssh/key.pem\" }], \"transport\": {\"type\": \"s3\", \"bucket_url\": \"s3://test\"}, \"assets\": [{ \"type\": \"file\", \"src\": \"local\", \"dst\": \"remote\", \"src_path\": \"package\", \"dst_path\": \"/etc/package\" }]}",
			isNotNil,
			isNil},
	}

	for _, test := range tests {
		t.Run(test.json, func(s *testing.T) {
			manifest, err := ParseManifest([]byte(test.json))
			if e := test.manifestFunc(manifest); e != nil {
				s.Errorf("invalid manifest: %v", e)
			}
			if e := test.errFunc(err); e != nil {
				s.Errorf("invalid manifest load error: %v", e)
			}
		})
	}
}
