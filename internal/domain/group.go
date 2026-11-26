package domain

import "fmt"

// ReasonCode is one value from the closed keeper-reason set.
// An absent recommendation is not a reason code; callers use the bool
// from ExactGroup.Recommendation.
type ReasonCode string

const (
	ReasonLibrarySurvivor    ReasonCode = "library_survivor"
	ReasonPlainName          ReasonCode = "plain_name"
	ReasonNewerMtimeTiebreak ReasonCode = "newer_mtime_tiebreak"
)

// Recommendation is a delete suggestion for an exact group: the keeper path,
// the paths suggested for Trash, and one reason code.
// Text is the human-readable reason. For newer_mtime_tiebreak it says the
// time was only a tie-break. An absent recommendation has no text and no code.
type Recommendation struct {
	Keeper string
	Trash  []string
	Reason ReasonCode
	Text   string
}

// NewRecommendation builds a recommendation. The reason must be one of the
// closed codes. An empty reason is rejected; absence is not a recommendation.
func NewRecommendation(keeper string, trash []string, reason ReasonCode) (Recommendation, error) {
	r := Recommendation{
		Keeper: keeper,
		Trash:  append([]string(nil), trash...),
		Reason: reason,
	}
	if err := r.validate(); err != nil {
		return Recommendation{}, err
	}
	return r, nil
}

func (r Recommendation) validate() error {
	switch r.Reason {
	case ReasonLibrarySurvivor, ReasonPlainName, ReasonNewerMtimeTiebreak:
	default:
		return fmt.Errorf("%w: %q", ErrReasonCode, r.Reason)
	}
	if r.Keeper == "" {
		return fmt.Errorf("%w: recommendation keeper", ErrEmptyPath)
	}
	return nil
}

func (r Recommendation) clone() Recommendation {
	r.Trash = append([]string(nil), r.Trash...)
	return r
}

// ExactGroup is two or more file refs that share a content ID.
// The recommendation is optional and starts absent.
type ExactGroup struct {
	members []FileRef
	rec     *Recommendation
}

// NewExactGroup rejects a group with fewer than two members, and a group
// whose members do not share one content ID.
func NewExactGroup(members []FileRef) (ExactGroup, error) {
	if len(members) < 2 {
		return ExactGroup{}, fmt.Errorf("%w: got %d", ErrExactGroupMembers, len(members))
	}
	id := members[0].ContentID
	for _, m := range members[1:] {
		if m.ContentID != id {
			return ExactGroup{}, ErrExactGroupContent
		}
	}
	return ExactGroup{members: cloneFiles(members)}, nil
}

// Members returns a copy of the group members.
func (g ExactGroup) Members() []FileRef {
	return cloneFiles(g.members)
}

// Recommendation returns the delete suggestion. The bool is false when the
// group has none. Absence is not a recommendation with an empty reason.
func (g ExactGroup) Recommendation() (Recommendation, bool) {
	if g.rec == nil {
		return Recommendation{}, false
	}
	return g.rec.clone(), true
}

// WithRecommendation returns a copy of the group that stores rec.
func (g ExactGroup) WithRecommendation(rec Recommendation) (ExactGroup, error) {
	if err := rec.validate(); err != nil {
		return ExactGroup{}, err
	}
	stored := rec.clone()
	g.rec = &stored
	return g, nil
}

// SimilarGroup is two or more file refs matched by a similarity strategy
// whose content IDs all differ. It never stores a delete recommendation.
type SimilarGroup struct {
	members []FileRef
}

// NewSimilarGroup rejects fewer than two members, and members that share a
// content ID. Byte-identical files belong in an exact group only.
func NewSimilarGroup(members []FileRef) (SimilarGroup, error) {
	if len(members) < 2 {
		return SimilarGroup{}, fmt.Errorf("%w: got %d", ErrSimilarGroupMembers, len(members))
	}
	seen := make(map[ContentID]struct{}, len(members))
	for _, m := range members {
		if m.ContentID == "" {
			return SimilarGroup{}, fmt.Errorf("%w: missing content ID for %s", ErrSimilarGroupContent, m.Path)
		}
		if _, ok := seen[m.ContentID]; ok {
			return SimilarGroup{}, ErrSimilarGroupContent
		}
		seen[m.ContentID] = struct{}{}
	}
	return SimilarGroup{members: cloneFiles(members)}, nil
}

// Members returns a copy of the group members.
func (g SimilarGroup) Members() []FileRef {
	return cloneFiles(g.members)
}

// Recommendation is always absent. A similar group has no delete suggestion.
func (SimilarGroup) Recommendation() (Recommendation, bool) {
	return Recommendation{}, false
}

// SetRecommendation rejects the suggestion. There is no field to store it.
func (SimilarGroup) SetRecommendation(Recommendation) error {
	return ErrSimilarRecommendation
}

// GroupExact returns one exact group for each content ID shared by two or
// more readable files, in the order those IDs first appear. A content ID
// that appears once is omitted. An empty content ID is not a group.
func GroupExact(files []FileRef) ([]ExactGroup, error) {
	var order []ContentID
	buckets := make(map[ContentID][]FileRef)
	for _, file := range files {
		if !file.Readable || file.ContentID == "" {
			continue
		}
		if _, ok := buckets[file.ContentID]; !ok {
			order = append(order, file.ContentID)
		}
		buckets[file.ContentID] = append(buckets[file.ContentID], file)
	}
	var groups []ExactGroup
	for _, id := range order {
		members := buckets[id]
		if len(members) < 2 {
			continue
		}
		group, err := NewExactGroup(members)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func cloneFiles(in []FileRef) []FileRef {
	out := make([]FileRef, len(in))
	copy(out, in)
	return out
}
