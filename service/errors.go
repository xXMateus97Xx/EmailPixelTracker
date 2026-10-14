package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrValidation      = errors.New("validation failed")
	// ErrImpersonationRequired is returned when a SuperAdmin accesses
	// tenant-scoped data without impersonating a tenant.
	ErrImpersonationRequired = errors.New("impersonation required to access tenant data")
	// ErrTenantInactive is returned for any operation of a user whose tenant is
	// inactive, and for writes of a SuperAdmin impersonating an inactive tenant.
	ErrTenantInactive = errors.New("tenant is inactive")
)

// Reasons why a pixel view is not tracked. All of them match ErrNotFound, which
// is what the public caller sees; the reason is only for logs and metrics.
var (
	ErrCampaignNotFound       = fmt.Errorf("%w: campaign does not exist", ErrNotFound)
	ErrCampaignInactive       = fmt.Errorf("%w: campaign is inactive", ErrNotFound)
	ErrCampaignExpired        = fmt.Errorf("%w: campaign has expired", ErrNotFound)
	ErrCampaignTenantInactive = fmt.Errorf("%w: tenant of the campaign is inactive", ErrNotFound)
)

type FieldError struct {
	Field   string
	Message string
}

// ValidationError matches ErrValidation with errors.Is.
type ValidationError struct {
	Errors []FieldError
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Errors))
	for i, fe := range e.Errors {
		msgs[i] = fe.Field + ": " + fe.Message
	}
	return ErrValidation.Error() + ": " + strings.Join(msgs, "; ")
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

type validator struct {
	errors []FieldError
}

func (v *validator) add(field, message string) {
	v.errors = append(v.errors, FieldError{Field: field, Message: message})
}

func (v *validator) check(ok bool, field, message string) {
	if !ok {
		v.add(field, message)
	}
}

func (v *validator) err() error {
	if len(v.errors) == 0 {
		return nil
	}
	return &ValidationError{Errors: v.errors}
}

// mapRepoError converts repository errors to service errors.
func mapRepoError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrDuplicate):
		return ErrConflict
	default:
		return err
	}
}
