package types

import (
	"errors"
	"regexp"
	"strings"
)

type UserRole string

const (
	UserRoleUser   UserRole = "user"
	UserRoleAdmin  UserRole = "admin"
	UserRoleEditor UserRole = "editor"
)

type AccountStatus string

const (
	AccountStatusActive      AccountStatus = "active"
	AccountStatusSuspended   AccountStatus = "suspended"
	AccountStatusDeactivated AccountStatus = "deactivated"
)

type VerificationStatus string

const (
	VerificationStatusPending  VerificationStatus = "pending"
	VerificationStatusApproved VerificationStatus = "approved"
	VerificationStatusRejected VerificationStatus = "rejected"
)

type DocumentType string

const (
	DocumentTypeRegistrationCertificate DocumentType = "registration_certificate"
	DocumentTypeTaxID                   DocumentType = "tax_id"
	DocumentTypeProofOfAddress          DocumentType = "proof_of_address"
	DocumentTypeIDDocument              DocumentType = "id_document"
	DocumentTypeOther                   DocumentType = "other"
)

type DocumentStatus string

const (
	DocumentStatusPending  DocumentStatus = "pending"
	DocumentStatusApproved DocumentStatus = "approved"
	DocumentStatusRejected DocumentStatus = "rejected"
)

var qkIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9._-]*[a-z0-9])?\.qk$`)

func NormalizeQKID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("qkId is required")
	}

	value = strings.ToLower(value)
	if !strings.HasSuffix(value, ".qk") {
		value = value + ".qk"
	}

	if !qkIDPattern.MatchString(value) {
		return "", errors.New("invalid qkId format")
	}

	return value, nil
}
