package publishing_test

import (
	"testing"

	"portfolio/internal/publishing"
)

func TestTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    publishing.SubmissionState
		to      publishing.SubmissionState
		actor   publishing.ActorKind
		wantErr bool
	}{
		{"agent starts build", publishing.Draft, publishing.Building, publishing.ActorAgent, false},
		{"system completes build", publishing.Building, publishing.PreviewReady, publishing.ActorSystem, false},
		{"system records failed build", publishing.Building, publishing.BuildFailed, publishing.ActorSystem, false},
		{"agent revises failed build", publishing.BuildFailed, publishing.Draft, publishing.ActorAgent, false},
		{"agent submits review", publishing.PreviewReady, publishing.InReview, publishing.ActorAgent, false},
		{"agent revises preview", publishing.PreviewReady, publishing.Draft, publishing.ActorAgent, false},
		{"agent cannot approve", publishing.InReview, publishing.Approved, publishing.ActorAgent, true},
		{"owner approves", publishing.InReview, publishing.Approved, publishing.ActorOwner, false},
		{"approved publishes", publishing.Approved, publishing.Published, publishing.ActorSystem, false},
		{"review changes requested", publishing.InReview, publishing.ChangesRequested, publishing.ActorOwner, false},
		{"agent revises requested changes", publishing.ChangesRequested, publishing.Draft, publishing.ActorAgent, false},
		{"owner rejects", publishing.InReview, publishing.Rejected, publishing.ActorOwner, false},
		{"published archives", publishing.Published, publishing.Archived, publishing.ActorOwner, false},
		{"owner cannot publish directly", publishing.Approved, publishing.Published, publishing.ActorOwner, true},
		{"cannot skip preview", publishing.Building, publishing.InReview, publishing.ActorSystem, true},
		{"unknown state rejected", publishing.SubmissionState("UNKNOWN"), publishing.Draft, publishing.ActorAgent, true},
		{"unknown actor rejected", publishing.Draft, publishing.Building, publishing.ActorKind("UNKNOWN"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := publishing.Transition(tt.from, tt.to, tt.actor)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Transition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubmissionReviewedArtifact(t *testing.T) {
	submission := publishing.Submission{
		ArtifactSHA256: "sha256:reviewed",
	}

	got := submission.ReviewedArtifact()
	if got.SHA256 != submission.ArtifactSHA256 {
		t.Fatalf("ReviewedArtifact().SHA256 = %q, want %q", got.SHA256, submission.ArtifactSHA256)
	}
}
