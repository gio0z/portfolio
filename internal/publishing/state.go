package publishing

import "fmt"

// SubmissionState is a submission's position in the publishing workflow.
type SubmissionState string

const (
	Draft            SubmissionState = "DRAFT"
	Building         SubmissionState = "BUILDING"
	BuildFailed      SubmissionState = "BUILD_FAILED"
	PreviewReady     SubmissionState = "PREVIEW_READY"
	InReview         SubmissionState = "IN_REVIEW"
	ChangesRequested SubmissionState = "CHANGES_REQUESTED"
	Rejected         SubmissionState = "REJECTED"
	Approved         SubmissionState = "APPROVED"
	Published        SubmissionState = "PUBLISHED"
	Archived         SubmissionState = "ARCHIVED"
)

type transition struct {
	from  SubmissionState
	to    SubmissionState
	actor ActorKind
}

var allowedTransitions = map[transition]struct{}{
	{Draft, Building, ActorAgent}:            {},
	{Building, PreviewReady, ActorSystem}:    {},
	{Building, BuildFailed, ActorSystem}:     {},
	{BuildFailed, Draft, ActorAgent}:         {},
	{PreviewReady, InReview, ActorAgent}:     {},
	{PreviewReady, Draft, ActorAgent}:        {},
	{InReview, Approved, ActorOwner}:         {},
	{InReview, ChangesRequested, ActorOwner}: {},
	{InReview, Rejected, ActorOwner}:         {},
	{ChangesRequested, Draft, ActorAgent}:    {},
	{Approved, Published, ActorSystem}:       {},
	{Published, Archived, ActorOwner}:        {},
}

// Transition verifies that actor may move a submission between the given states.
func Transition(current, next SubmissionState, actor ActorKind) error {
	if _, ok := allowedTransitions[transition{from: current, to: next, actor: actor}]; !ok {
		return fmt.Errorf("publishing: transition %s -> %s is not allowed for %s", current, next, actor)
	}
	return nil
}
