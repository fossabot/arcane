package build

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	depotbuild "github.com/depot/depot-go/build"
	depotmachine "github.com/depot/depot-go/machine"
	cliv1 "github.com/depot/depot-go/proto/depot/cli/v1"
	"github.com/getarcaneapp/arcane/types/image"
	"github.com/moby/buildkit/client"
)

type BuildSettings struct {
	BuildkitEndpoint string
	DepotProjectId   string
	DepotToken       string
}

type SettingsProvider interface {
	BuildSettings() BuildSettings
}

type BuildSession struct {
	Client *client.Client
	Close  func(buildErr error) error
}

type BuildProvider interface {
	Name() string
	NewSession(ctx context.Context, req image.BuildRequest) (*BuildSession, error)
}

type LocalBuildKitProvider struct {
	settings SettingsProvider
}

func NewLocalBuildKitProvider(settings SettingsProvider) *LocalBuildKitProvider {
	return &LocalBuildKitProvider{settings: settings}
}

func (p *LocalBuildKitProvider) Name() string {
	return "local"
}

func (p *LocalBuildKitProvider) NewSession(ctx context.Context, _ image.BuildRequest) (*BuildSession, error) {
	if p.settings == nil {
		return nil, errors.New("settings provider not available")
	}

	settings := p.settings.BuildSettings()
	endpoint := strings.TrimSpace(settings.BuildkitEndpoint)
	if endpoint == "" {
		endpoint = "unix:///run/buildkit/buildkitd.sock"
	}

	bk, err := client.New(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to BuildKit: %w", err)
	}

	return &BuildSession{
		Client: bk,
		Close: func(buildErr error) error {
			_ = buildErr
			return bk.Close()
		},
	}, nil
}

type DepotBuildKitProvider struct {
	settings SettingsProvider
}

func NewDepotBuildKitProvider(settings SettingsProvider) *DepotBuildKitProvider {
	return &DepotBuildKitProvider{settings: settings}
}

func (p *DepotBuildKitProvider) Name() string {
	return "depot"
}

func (p *DepotBuildKitProvider) NewSession(ctx context.Context, req image.BuildRequest) (*BuildSession, error) {
	if p.settings == nil {
		return nil, errors.New("settings provider not available")
	}

	settings := p.settings.BuildSettings()
	projectID := strings.TrimSpace(settings.DepotProjectId)
	token := strings.TrimSpace(settings.DepotToken)
	if projectID == "" || token == "" {
		return nil, errors.New("Depot project ID and token are required")
	}

	buildReq := &cliv1.CreateBuildRequest{ProjectId: projectID}
	build, err := depotbuild.NewBuild(ctx, buildReq, token)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Depot build: %w", err)
	}

	arch := selectDepotArch(req.Platforms)
	machine, err := depotmachine.Acquire(ctx, build.ID, build.Token, arch)
	if err != nil {
		build.Finish(err)
		return nil, fmt.Errorf("failed to acquire Depot BuildKit machine: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	bk, err := machine.Connect(connectCtx)
	if err != nil {
		_ = machine.Release()
		build.Finish(err)
		return nil, fmt.Errorf("failed to connect to Depot BuildKit: %w", err)
	}

	return &BuildSession{
		Client: bk,
		Close: func(buildErr error) error {
			build.Finish(buildErr)
			releaseErr := machine.Release()
			closeErr := bk.Close()
			return errors.Join(releaseErr, closeErr)
		},
	}, nil
}

func selectDepotArch(platforms []string) string {
	for _, platform := range platforms {
		p := strings.ToLower(strings.TrimSpace(platform))
		switch {
		case strings.Contains(p, "arm64"):
			return "arm64"
		case strings.Contains(p, "amd64"):
			return "amd64"
		}
	}

	if runtime.GOARCH == "arm64" {
		return "arm64"
	}
	return "amd64"
}
