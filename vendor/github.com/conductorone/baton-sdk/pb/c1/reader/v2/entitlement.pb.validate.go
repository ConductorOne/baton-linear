// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: c1/reader/v2/entitlement.proto

package v2

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on
// EntitlementsReaderServiceGetEntitlementRequest with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *EntitlementsReaderServiceGetEntitlementRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on
// EntitlementsReaderServiceGetEntitlementRequest with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in
// EntitlementsReaderServiceGetEntitlementRequestMultiError, or nil if none found.
func (m *EntitlementsReaderServiceGetEntitlementRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *EntitlementsReaderServiceGetEntitlementRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if l := len(m.GetEntitlementId()); l < 1 || l > 1024 {
		err := EntitlementsReaderServiceGetEntitlementRequestValidationError{
			field:  "EntitlementId",
			reason: "value length must be between 1 and 1024 bytes, inclusive",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	for idx, item := range m.GetAnnotations() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, EntitlementsReaderServiceGetEntitlementRequestValidationError{
						field:  fmt.Sprintf("Annotations[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, EntitlementsReaderServiceGetEntitlementRequestValidationError{
						field:  fmt.Sprintf("Annotations[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return EntitlementsReaderServiceGetEntitlementRequestValidationError{
					field:  fmt.Sprintf("Annotations[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return EntitlementsReaderServiceGetEntitlementRequestMultiError(errors)
	}

	return nil
}

// EntitlementsReaderServiceGetEntitlementRequestMultiError is an error
// wrapping multiple validation errors returned by
// EntitlementsReaderServiceGetEntitlementRequest.ValidateAll() if the
// designated constraints aren't met.
type EntitlementsReaderServiceGetEntitlementRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m EntitlementsReaderServiceGetEntitlementRequestMultiError) Error() string {
	msgs := make([]string, 0, len(m))
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m EntitlementsReaderServiceGetEntitlementRequestMultiError) AllErrors() []error { return m }

// EntitlementsReaderServiceGetEntitlementRequestValidationError is the
// validation error returned by
// EntitlementsReaderServiceGetEntitlementRequest.Validate if the designated
// constraints aren't met.
type EntitlementsReaderServiceGetEntitlementRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e EntitlementsReaderServiceGetEntitlementRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e EntitlementsReaderServiceGetEntitlementRequestValidationError) Reason() string {
	return e.reason
}

// Cause function returns cause value.
func (e EntitlementsReaderServiceGetEntitlementRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e EntitlementsReaderServiceGetEntitlementRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e EntitlementsReaderServiceGetEntitlementRequestValidationError) ErrorName() string {
	return "EntitlementsReaderServiceGetEntitlementRequestValidationError"
}

// Error satisfies the builtin error interface
func (e EntitlementsReaderServiceGetEntitlementRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sEntitlementsReaderServiceGetEntitlementRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = EntitlementsReaderServiceGetEntitlementRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = EntitlementsReaderServiceGetEntitlementRequestValidationError{}

// Validate checks the field values on
// EntitlementsReaderServiceGetEntitlementResponse with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *EntitlementsReaderServiceGetEntitlementResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on
// EntitlementsReaderServiceGetEntitlementResponse with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in
// EntitlementsReaderServiceGetEntitlementResponseMultiError, or nil if none found.
func (m *EntitlementsReaderServiceGetEntitlementResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *EntitlementsReaderServiceGetEntitlementResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetEntitlement()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, EntitlementsReaderServiceGetEntitlementResponseValidationError{
					field:  "Entitlement",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, EntitlementsReaderServiceGetEntitlementResponseValidationError{
					field:  "Entitlement",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetEntitlement()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return EntitlementsReaderServiceGetEntitlementResponseValidationError{
				field:  "Entitlement",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return EntitlementsReaderServiceGetEntitlementResponseMultiError(errors)
	}

	return nil
}

// EntitlementsReaderServiceGetEntitlementResponseMultiError is an error
// wrapping multiple validation errors returned by
// EntitlementsReaderServiceGetEntitlementResponse.ValidateAll() if the
// designated constraints aren't met.
type EntitlementsReaderServiceGetEntitlementResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m EntitlementsReaderServiceGetEntitlementResponseMultiError) Error() string {
	msgs := make([]string, 0, len(m))
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m EntitlementsReaderServiceGetEntitlementResponseMultiError) AllErrors() []error { return m }

// EntitlementsReaderServiceGetEntitlementResponseValidationError is the
// validation error returned by
// EntitlementsReaderServiceGetEntitlementResponse.Validate if the designated
// constraints aren't met.
type EntitlementsReaderServiceGetEntitlementResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e EntitlementsReaderServiceGetEntitlementResponseValidationError) Field() string {
	return e.field
}

// Reason function returns reason value.
func (e EntitlementsReaderServiceGetEntitlementResponseValidationError) Reason() string {
	return e.reason
}

// Cause function returns cause value.
func (e EntitlementsReaderServiceGetEntitlementResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e EntitlementsReaderServiceGetEntitlementResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e EntitlementsReaderServiceGetEntitlementResponseValidationError) ErrorName() string {
	return "EntitlementsReaderServiceGetEntitlementResponseValidationError"
}

// Error satisfies the builtin error interface
func (e EntitlementsReaderServiceGetEntitlementResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sEntitlementsReaderServiceGetEntitlementResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = EntitlementsReaderServiceGetEntitlementResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = EntitlementsReaderServiceGetEntitlementResponseValidationError{}
