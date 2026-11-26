package domain

import "errors"

// Sentinel errors for domain checks. Callers compare them with errors.Is.
var (
	ErrExactGroupMembers = errors.New("exact group needs at least two members")
	ErrExactGroupContent = errors.New("exact group members must share a content ID")

	ErrSimilarGroupMembers   = errors.New("similar group needs at least two members")
	ErrSimilarGroupContent   = errors.New("similar group members must have different content IDs")
	ErrSimilarRecommendation = errors.New("similar group cannot store a delete recommendation")

	ErrDecisionChoice = errors.New("decision choice must be keep, delete, or skip")
	ErrEmptyPath      = errors.New("path is empty")

	ErrReasonCode = errors.New("unknown recommendation reason")

	ErrUnconfirmedDelete = errors.New("plan trashes a path with no delete decision")
	ErrLastCopy          = errors.New("plan trashes every copy of a content ID with no library entry")

	ErrMoveOutsideRoot = errors.New("layout move destination escapes the scan root")

	ErrSessionNotOpen = errors.New("session is not open")
)
