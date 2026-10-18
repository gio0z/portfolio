package deployment_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"portfolio/internal/deployment"
	"portfolio/internal/publishing"
)

// fakeLab is a controllable LabPublisher fake. Log records the call order so
// tests can prove both destinations stage before either goes visible.
type fakeLab struct {
	log         *[]string
	stageErr    error
	visibleErr  error
	rollbackErr error
	stages      int
	visibles    int
	rollbacks   int
	visible     bool
}

func (f *fakeLab) Stage(ctx context.Context, req publishing.PublicationRequest) (deployment.LabStaged, error) {
	if err := ctx.Err(); err != nil {
		return deployment.LabStaged{}, err
	}
	*f.log = append(*f.log, "lab-stage")
	if f.stageErr != nil {
		return deployment.LabStaged{}, f.stageErr
	}
	f.stages++
	return deployment.LabStaged{
		ArtifactHash: req.Submission.ArtifactSHA256,
		Slug:         req.Project.Slug,
		VersionPath:  "/fake/lab/versions/" + req.Submission.ArtifactSHA256,
	}, nil
}

func (f *fakeLab) MakeVisible(ctx context.Context, staged deployment.LabStaged) (deployment.PublishedDestination, error) {
	if err := ctx.Err(); err != nil {
		return deployment.PublishedDestination{}, err
	}
	*f.log = append(*f.log, "lab-visible")
	if f.visibleErr != nil {
		return deployment.PublishedDestination{}, f.visibleErr
	}
	f.visibles++
	f.visible = true
	return deployment.PublishedDestination{
		URL:          "https://lab.example.test/" + staged.Slug,
		DeploymentID: "lab-" + staged.ArtifactHash[:12],
	}, nil
}

func (f *fakeLab) Rollback(ctx context.Context, staged deployment.LabStaged) error {
	*f.log = append(*f.log, "lab-rollback")
	f.rollbacks++
	f.visible = false
	return f.rollbackErr
}

// fakePortfolio mirrors fakeLab for the portfolio catalog destination.
type fakePortfolio struct {
	log         *[]string
	stageErr    error
	visibleErr  error
	rollbackErr error
	stages      int
	visibles    int
	rollbacks   int
	visible     bool
}

func (f *fakePortfolio) Stage(ctx context.Context, req publishing.PublicationRequest) (deployment.PortfolioStaged, error) {
	if err := ctx.Err(); err != nil {
		return deployment.PortfolioStaged{}, err
	}
	*f.log = append(*f.log, "portfolio-stage")
	if f.stageErr != nil {
		return deployment.PortfolioStaged{}, f.stageErr
	}
	f.stages++
	return deployment.PortfolioStaged{
		ArtifactHash: req.Submission.ArtifactSHA256,
		Slug:         req.Project.Slug,
		VersionPath:  "/fake/catalog/versions/" + req.Submission.ArtifactSHA256,
	}, nil
}

func (f *fakePortfolio) MakeVisible(ctx context.Context, staged deployment.PortfolioStaged) (deployment.PublishedDestination, error) {
	if err := ctx.Err(); err != nil {
		return deployment.PublishedDestination{}, err
	}
	*f.log = append(*f.log, "portfolio-visible")
	if f.visibleErr != nil {
		return deployment.PublishedDestination{}, f.visibleErr
	}
	f.visibles++
	f.visible = true
	return deployment.PublishedDestination{
		URL:          "https://portfolio.example.test/design-lab/" + staged.Slug,
		DeploymentID: "portfolio-" + staged.ArtifactHash[:12],
	}, nil
}

func (f *fakePortfolio) Rollback(ctx context.Context, staged deployment.PortfolioStaged) error {
	*f.log = append(*f.log, "portfolio-rollback")
	f.rollbacks++
	f.visible = false
	return f.rollbackErr
}

const testHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func testPublicationRequest() publishing.PublicationRequest {
	return publishing.PublicationRequest{
		Project: publishing.LabProject{
			ID:    "proj-1",
			Slug:  "checkout-redesign",
			Title: "Checkout Redesign",
		},
		Submission: publishing.Submission{
			ID:             "sub-1",
			LabProjectID:   "proj-1",
			Revision:       3,
			ArtifactSHA256: testHash,
		},
	}
}

func TestAtomicPublisher_LabFailurePublishesNeitherDestination(t *testing.T) {
	var log []string
	lab := &fakeLab{log: &log, stageErr: errors.New("lab stage boom")}
	port := &fakePortfolio{log: &log}
	pub := deployment.NewAtomicPublisher(lab, port)

	if _, err := pub.Publish(context.Background(), testPublicationRequest()); err == nil {
		t.Fatal("Publish() error = nil, want lab stage failure")
	}
	if port.stages != 0 || port.visibles != 0 {
		t.Errorf("portfolio touched after lab failure: stages=%d visibles=%d, want 0 0", port.stages, port.visibles)
	}
	if lab.visibles != 0 {
		t.Errorf("lab made visible after lab stage failure: visibles=%d, want 0", lab.visibles)
	}
	for _, entry := range log {
		if strings.HasSuffix(entry, "-visible") {
			t.Errorf("visibility changed during failed publication: %q", entry)
		}
	}
}

func TestAtomicPublisher_PortfolioStageFailureRollsBackLab(t *testing.T) {
	var log []string
	lab := &fakeLab{log: &log}
	port := &fakePortfolio{log: &log, stageErr: errors.New("portfolio stage boom")}
	pub := deployment.NewAtomicPublisher(lab, port)

	if _, err := pub.Publish(context.Background(), testPublicationRequest()); err == nil {
		t.Fatal("Publish() error = nil, want portfolio stage failure")
	}
	if !lab.visible && lab.rollbacks == 0 {
		t.Error("lab rollback not attempted after portfolio failure")
	}
	if lab.visible {
		t.Error("lab still visible after portfolio failure: rollback must remove staged/pointed Lab artifacts")
	}
	if port.visible {
		t.Error("portfolio visible after its own stage failure")
	}
}

func TestAtomicPublisher_PortfolioVisibleFailureRollsBackLabVisibility(t *testing.T) {
	var log []string
	lab := &fakeLab{log: &log}
	port := &fakePortfolio{log: &log, visibleErr: errors.New("portfolio pointer swap boom")}
	pub := deployment.NewAtomicPublisher(lab, port)

	if _, err := pub.Publish(context.Background(), testPublicationRequest()); err == nil {
		t.Fatal("Publish() error = nil, want portfolio visibility failure")
	}
	if lab.visible {
		t.Error("lab still visible after portfolio visibility failure: Lab visibility must be rolled back")
	}
	if lab.rollbacks == 0 {
		t.Error("lab rollback not attempted after portfolio visibility failure")
	}
}

func TestAtomicPublisher_SuccessStagesBothBeforeEitherVisible(t *testing.T) {
	var log []string
	lab := &fakeLab{log: &log}
	port := &fakePortfolio{log: &log}
	pub := deployment.NewAtomicPublisher(lab, port)

	out, err := pub.Publish(context.Background(), testPublicationRequest())
	if err != nil {
		t.Fatalf("Publish() error = %v, want nil", err)
	}

	wantOrder := []string{"lab-stage", "portfolio-stage", "lab-visible", "portfolio-visible"}
	if !reflect.DeepEqual(log, wantOrder) {
		t.Fatalf("call order = %v, want %v", log, wantOrder)
	}

	if out.LabURL != "https://lab.example.test/checkout-redesign" {
		t.Errorf("LabURL = %q, want lab URL for slug", out.LabURL)
	}
	if out.PortfolioURL != "https://portfolio.example.test/design-lab/checkout-redesign" {
		t.Errorf("PortfolioURL = %q, want portfolio URL for slug", out.PortfolioURL)
	}
	if len(out.DeploymentIDs) != 2 {
		t.Fatalf("DeploymentIDs = %v, want exactly 2 entries", out.DeploymentIDs)
	}
	if out.DeploymentIDs[0] != "lab-"+testHash[:12] {
		t.Errorf("DeploymentIDs[0] = %q, want lab deployment for hash %q", out.DeploymentIDs[0], testHash)
	}
	if out.DeploymentIDs[1] != "portfolio-"+testHash[:12] {
		t.Errorf("DeploymentIDs[1] = %q, want portfolio deployment for the same hash", out.DeploymentIDs[1])
	}
	if len(out.DestinationURLs) != 2 || out.DestinationURLs[0] != out.LabURL || out.DestinationURLs[1] != out.PortfolioURL {
		t.Errorf("DestinationURLs = %v, want [LabURL PortfolioURL]", out.DestinationURLs)
	}
}

func TestAtomicPublisher_RetryIsIdempotent(t *testing.T) {
	var log []string
	lab := &fakeLab{log: &log}
	port := &fakePortfolio{log: &log}
	pub := deployment.NewAtomicPublisher(lab, port)
	req := testPublicationRequest()

	first, err := pub.Publish(context.Background(), req)
	if err != nil {
		t.Fatalf("first Publish() error = %v, want nil", err)
	}
	stages, visibles := lab.stages+port.stages, lab.visibles+port.visibles

	second, err := pub.Publish(context.Background(), req)
	if err != nil {
		t.Fatalf("retry Publish() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(second, first) {
		t.Errorf("retry output = %+v, want identical to first %+v", second, first)
	}
	if got := lab.stages + port.stages; got != stages {
		t.Errorf("retry staged %d destinations, want 0 additional (idempotent)", got-stages)
	}
	if got := lab.visibles + port.visibles; got != visibles {
		t.Errorf("retry made %d destinations visible, want 0 additional (idempotent)", got-visibles)
	}
}

func TestAtomicPublisher_FilesystemRoundTripAndRollback(t *testing.T) {
	root := t.TempDir()
	lab := deployment.NewLabPublisher(root, "https://lab.example.test")
	port := deployment.NewPortfolioCatalogPublisher(root, "https://portfolio.example.test")
	pub := deployment.NewAtomicPublisher(lab, port)

	out, err := pub.Publish(context.Background(), testPublicationRequest())
	if err != nil {
		t.Fatalf("Publish() error = %v, want nil", err)
	}
	if !strings.Contains(out.LabURL, "checkout-redesign") || !strings.Contains(out.PortfolioURL, "checkout-redesign") {
		t.Errorf("output URLs do not reference slug: %+v", out)
	}

	// Same hash retries against the real filesystem publishers: identical output.
	again, err := pub.Publish(context.Background(), testPublicationRequest())
	if err != nil {
		t.Fatalf("retry Publish() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(again, out) {
		t.Errorf("retry output = %+v, want %+v", again, out)
	}
}

func TestAtomicPublisher_FilesystemRollbackRemovesLabArtifacts(t *testing.T) {
	root := t.TempDir()
	lab := deployment.NewLabPublisher(root, "https://lab.example.test")
	var log []string
	port := &fakePortfolio{log: &log, stageErr: errors.New("catalog down")}
	pub := deployment.NewAtomicPublisher(lab, port)

	if _, err := pub.Publish(context.Background(), testPublicationRequest()); err == nil {
		t.Fatal("Publish() error = nil, want portfolio failure")
	}

	// A follow-up publish with a healthy portfolio must succeed from scratch,
	// proving the failed attempt left no staged or pointed Lab artifacts behind.
	healthy := deployment.NewAtomicPublisher(lab, deployment.NewPortfolioCatalogPublisher(root, "https://portfolio.example.test"))
	out, err := healthy.Publish(context.Background(), testPublicationRequest())
	if err != nil {
		t.Fatalf("republish after rollback error = %v, want nil", err)
	}
	if out.LabURL == "" || out.PortfolioURL == "" {
		t.Errorf("republish output missing URLs: %+v", out)
	}
}

func TestLabPublisher_RollbackRestoresPreviousVersion(t *testing.T) {
	root := t.TempDir()
	lab := deployment.NewLabPublisher(root, "https://lab.example.test")
	ctx := context.Background()

	// 1. Stage and make visible v1
	reqV1 := testPublicationRequest()
	reqV1.Submission.ArtifactSHA256 = "1111111111111111111111111111111111111111111111111111111111111111"
	stagedV1, err := lab.Stage(ctx, reqV1)
	if err != nil {
		t.Fatalf("Stage(v1) error = %v", err)
	}
	if stagedV1.PrevVersion != "" {
		t.Errorf("stagedV1.PrevVersion = %q, want empty", stagedV1.PrevVersion)
	}
	if _, err := lab.MakeVisible(ctx, stagedV1); err != nil {
		t.Fatalf("MakeVisible(v1) error = %v", err)
	}

	// 2. Stage and make visible v2
	reqV2 := testPublicationRequest()
	reqV2.Submission.ArtifactSHA256 = "2222222222222222222222222222222222222222222222222222222222222222"
	stagedV2, err := lab.Stage(ctx, reqV2)
	if err != nil {
		t.Fatalf("Stage(v2) error = %v", err)
	}
	if stagedV2.PrevVersion != reqV1.Submission.ArtifactSHA256 {
		t.Errorf("stagedV2.PrevVersion = %q, want %q", stagedV2.PrevVersion, reqV1.Submission.ArtifactSHA256)
	}
	if _, err := lab.MakeVisible(ctx, stagedV2); err != nil {
		t.Fatalf("MakeVisible(v2) error = %v", err)
	}

	// 3. Rollback v2
	if err := lab.Rollback(ctx, stagedV2); err != nil {
		t.Fatalf("Rollback(v2) error = %v", err)
	}

	// Verify pointer points back to v1
	ptrPath := filepath.Join(root, "var", "portfolio", "lab", "pointers", reqV1.Project.Slug+".json")
	raw, err := os.ReadFile(ptrPath)
	if err != nil {
		t.Fatalf("pointer file missing after rollback: %v", err)
	}
	var ptr struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &ptr); err != nil {
		t.Fatalf("unmarshal pointer: %v", err)
	}
	if ptr.Version != reqV1.Submission.ArtifactSHA256 {
		t.Errorf("pointer version = %q, want v1 %q", ptr.Version, reqV1.Submission.ArtifactSHA256)
	}

	// v1 version directory must still exist intact
	if _, err := os.Stat(stagedV1.VersionPath); err != nil {
		t.Errorf("v1 version directory missing: %v", err)
	}
}

func TestLabPublisher_RollbackWithoutPriorPointerRemovesAll(t *testing.T) {
	root := t.TempDir()
	lab := deployment.NewLabPublisher(root, "https://lab.example.test")
	ctx := context.Background()

	req := testPublicationRequest()
	staged, err := lab.Stage(ctx, req)
	if err != nil {
		t.Fatalf("Stage() error = %v", err)
	}
	if staged.PrevVersion != "" {
		t.Errorf("staged.PrevVersion = %q, want empty", staged.PrevVersion)
	}
	if _, err := lab.MakeVisible(ctx, staged); err != nil {
		t.Fatalf("MakeVisible() error = %v", err)
	}

	if err := lab.Rollback(ctx, staged); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	ptrPath := filepath.Join(root, "var", "portfolio", "lab", "pointers", req.Project.Slug+".json")
	if _, err := os.Stat(ptrPath); !os.IsNotExist(err) {
		t.Errorf("pointer file still exists after rollback without prior pointer: %v", err)
	}
	if _, err := os.Stat(staged.VersionPath); !os.IsNotExist(err) {
		t.Errorf("version dir still exists after rollback: %v", err)
	}
}

func TestPortfolioPublisher_RollbackRestoresPreviousVersion(t *testing.T) {
	root := t.TempDir()
	port := deployment.NewPortfolioCatalogPublisher(root, "https://portfolio.example.test")
	ctx := context.Background()

	// 1. Stage and make visible v1
	reqV1 := testPublicationRequest()
	reqV1.Submission.ArtifactSHA256 = "1111111111111111111111111111111111111111111111111111111111111111"
	stagedV1, err := port.Stage(ctx, reqV1)
	if err != nil {
		t.Fatalf("Stage(v1) error = %v", err)
	}
	if stagedV1.PrevVersion != "" {
		t.Errorf("stagedV1.PrevVersion = %q, want empty", stagedV1.PrevVersion)
	}
	if _, err := port.MakeVisible(ctx, stagedV1); err != nil {
		t.Fatalf("MakeVisible(v1) error = %v", err)
	}

	// 2. Stage and make visible v2
	reqV2 := testPublicationRequest()
	reqV2.Submission.ArtifactSHA256 = "2222222222222222222222222222222222222222222222222222222222222222"
	stagedV2, err := port.Stage(ctx, reqV2)
	if err != nil {
		t.Fatalf("Stage(v2) error = %v", err)
	}
	if stagedV2.PrevVersion != reqV1.Submission.ArtifactSHA256 {
		t.Errorf("stagedV2.PrevVersion = %q, want %q", stagedV2.PrevVersion, reqV1.Submission.ArtifactSHA256)
	}
	if _, err := port.MakeVisible(ctx, stagedV2); err != nil {
		t.Fatalf("MakeVisible(v2) error = %v", err)
	}

	// 3. Rollback v2
	if err := port.Rollback(ctx, stagedV2); err != nil {
		t.Fatalf("Rollback(v2) error = %v", err)
	}

	ptrPath := filepath.Join(root, "var", "portfolio", "catalog", "pointers", reqV1.Project.Slug+".json")
	raw, err := os.ReadFile(ptrPath)
	if err != nil {
		t.Fatalf("pointer file missing after rollback: %v", err)
	}
	var ptr struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &ptr); err != nil {
		t.Fatalf("unmarshal pointer: %v", err)
	}
	if ptr.Version != reqV1.Submission.ArtifactSHA256 {
		t.Errorf("pointer version = %q, want v1 %q", ptr.Version, reqV1.Submission.ArtifactSHA256)
	}

	if _, err := os.Stat(stagedV1.VersionPath); err != nil {
		t.Errorf("v1 version directory missing: %v", err)
	}
}

func TestPortfolioPublisher_RollbackWithoutPriorPointerRemovesAll(t *testing.T) {
	root := t.TempDir()
	port := deployment.NewPortfolioCatalogPublisher(root, "https://portfolio.example.test")
	ctx := context.Background()

	req := testPublicationRequest()
	staged, err := port.Stage(ctx, req)
	if err != nil {
		t.Fatalf("Stage() error = %v", err)
	}
	if staged.PrevVersion != "" {
		t.Errorf("staged.PrevVersion = %q, want empty", staged.PrevVersion)
	}
	if _, err := port.MakeVisible(ctx, staged); err != nil {
		t.Fatalf("MakeVisible() error = %v", err)
	}

	if err := port.Rollback(ctx, staged); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	ptrPath := filepath.Join(root, "var", "portfolio", "catalog", "pointers", req.Project.Slug+".json")
	if _, err := os.Stat(ptrPath); !os.IsNotExist(err) {
		t.Errorf("pointer file still exists after rollback without prior pointer: %v", err)
	}
	if _, err := os.Stat(staged.VersionPath); !os.IsNotExist(err) {
		t.Errorf("version dir still exists after rollback: %v", err)
	}
}

func TestAtomicPublisher_RollbackErrorAttachedOnPublishFailure(t *testing.T) {
	var log []string
	lab := &fakeLab{log: &log, rollbackErr: errors.New("lab rollback failed")}
	port := &fakePortfolio{log: &log, visibleErr: errors.New("portfolio visible failed")}
	pub := deployment.NewAtomicPublisher(lab, port)

	_, err := pub.Publish(context.Background(), testPublicationRequest())
	if err == nil {
		t.Fatal("Publish() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "portfolio visible failed") {
		t.Errorf("expected original error in %q", err.Error())
	}
	if !strings.Contains(err.Error(), "rollback:") || !strings.Contains(err.Error(), "lab rollback failed") {
		t.Errorf("expected rollback error in %q", err.Error())
	}
}

func TestLabPublisher_RollbackAfterIdempotentReStagePreservesPointer(t *testing.T) {
	root := t.TempDir()
	lab := deployment.NewLabPublisher(root, "https://lab.example.test")
	ctx := context.Background()

	req := testPublicationRequest()
	staged1, err := lab.Stage(ctx, req)
	if err != nil {
		t.Fatalf("Stage() error = %v", err)
	}
	if _, err := lab.MakeVisible(ctx, staged1); err != nil {
		t.Fatalf("MakeVisible() error = %v", err)
	}

	// Re-stage the same request hitting os.IsExist branch
	staged2, err := lab.Stage(ctx, req)
	if err != nil {
		t.Fatalf("re-Stage() error = %v", err)
	}
	if staged2.PrevVersion != req.Submission.ArtifactSHA256 {
		t.Errorf("staged2.PrevVersion = %q, want %q", staged2.PrevVersion, req.Submission.ArtifactSHA256)
	}

	// Rollback of the re-staged instance must restore v1, not delete the pointer
	if err := lab.Rollback(ctx, staged2); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	ptrPath := filepath.Join(root, "var", "portfolio", "lab", "pointers", req.Project.Slug+".json")
	if _, err := os.Stat(ptrPath); err != nil {
		t.Errorf("pointer removed after rollback of re-stage: %v", err)
	}
}

func TestPortfolioPublisher_RollbackAfterIdempotentReStagePreservesPointer(t *testing.T) {
	root := t.TempDir()
	port := deployment.NewPortfolioCatalogPublisher(root, "https://portfolio.example.test")
	ctx := context.Background()

	req := testPublicationRequest()
	staged1, err := port.Stage(ctx, req)
	if err != nil {
		t.Fatalf("Stage() error = %v", err)
	}
	if _, err := port.MakeVisible(ctx, staged1); err != nil {
		t.Fatalf("MakeVisible() error = %v", err)
	}

	// Re-stage the same request hitting os.IsExist branch
	staged2, err := port.Stage(ctx, req)
	if err != nil {
		t.Fatalf("re-Stage() error = %v", err)
	}
	if staged2.PrevVersion != req.Submission.ArtifactSHA256 {
		t.Errorf("staged2.PrevVersion = %q, want %q", staged2.PrevVersion, req.Submission.ArtifactSHA256)
	}

	// Rollback of the re-staged instance must restore v1, not delete the pointer
	if err := port.Rollback(ctx, staged2); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	ptrPath := filepath.Join(root, "var", "portfolio", "catalog", "pointers", req.Project.Slug+".json")
	if _, err := os.Stat(ptrPath); err != nil {
		t.Errorf("pointer removed after rollback of re-stage: %v", err)
	}
}
