package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/getarcaneapp/arcane/backend/internal/utils/timeouts"
	arcane_build "github.com/getarcaneapp/arcane/backend/pkg/libarcane/libbuild"
	imagetypes "github.com/getarcaneapp/arcane/types/image"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session/auth/authprovider"
)

type BuildService struct {
	settings      *SettingsService
	dockerService *DockerClientService
	providers     map[string]arcane_build.BuildProvider
}

func NewBuildService(settings *SettingsService, dockerService *DockerClientService) *BuildService {
	settingsProvider := buildSettingsProvider{settings: settings}
	providers := map[string]arcane_build.BuildProvider{
		"local": arcane_build.NewLocalBuildKitProvider(settingsProvider),
		"depot": arcane_build.NewDepotBuildKitProvider(settingsProvider),
	}

	return &BuildService{
		settings:      settings,
		dockerService: dockerService,
		providers:     providers,
	}
}

type buildSettingsProvider struct {
	settings *SettingsService
}

func (p buildSettingsProvider) BuildSettings() arcane_build.BuildSettings {
	if p.settings == nil {
		return arcane_build.BuildSettings{}
	}
	settings := p.settings.GetSettingsConfig()
	return arcane_build.BuildSettings{
		BuildkitEndpoint: settings.BuildkitEndpoint.Value,
		DepotProjectId:   settings.DepotProjectId.Value,
		DepotToken:       settings.DepotToken.Value,
	}
}

func (s *BuildService) BuildImage(ctx context.Context, req imagetypes.BuildRequest, progressWriter io.Writer, serviceName string) (*imagetypes.BuildResult, error) {
	if s.settings == nil {
		return nil, errors.New("settings service not available")
	}

	if strings.TrimSpace(req.ContextDir) == "" {
		return nil, errors.New("contextDir is required")
	}

	providerName, provider, err := s.resolveProvider(req.Provider)
	if err != nil {
		return nil, err
	}

	settings := s.settings.GetSettingsConfig()
	buildCtx, cancel := timeouts.WithTimeout(ctx, settings.BuildTimeout.AsInt(), timeouts.DefaultBuildTimeout)
	defer cancel()

	req = normalizeBuildRequest(req, providerName)
	req.Tags = normalizeTags(req.Tags)

	if err := validateBuildRequest(req, providerName); err != nil {
		return nil, err
	}

	session, err := provider.NewSession(buildCtx, req)
	if err != nil {
		return nil, err
	}

	var buildErr error
	defer func() {
		if cerr := session.Close(buildErr); cerr != nil {
			slog.WarnContext(ctx, "build session close error", "provider", providerName, "error", cerr)
		}
	}()

	solveOpt, loadErrCh, err := s.buildSolveOpt(buildCtx, req)
	if err != nil {
		buildErr = err
		return nil, err
	}

	authProvider := authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
		ConfigFile: config.LoadDefaultConfigFile(os.Stderr),
	})
	solveOpt.Session = append(solveOpt.Session, authProvider)

	statusCh := make(chan *client.SolveStatus, 16)
	streamErrCh := make(chan error, 1)
	go func() {
		streamErrCh <- streamSolveStatus(buildCtx, statusCh, progressWriter, serviceName)
	}()

	writeProgressEvent(progressWriter, imagetypes.ProgressEvent{
		Type:    "build",
		Phase:   "begin",
		Service: serviceName,
		Status:  "build started",
	})

	resp, err := session.Client.Solve(buildCtx, nil, solveOpt, statusCh)
	buildErr = err

	if err != nil {
		writeProgressEvent(progressWriter, imagetypes.ProgressEvent{
			Type:    "build",
			Service: serviceName,
			Error:   err.Error(),
		})
		return nil, err
	}

	if streamErr := <-streamErrCh; streamErr != nil && !errors.Is(streamErr, context.Canceled) {
		slog.WarnContext(ctx, "build progress stream error", "provider", providerName, "error", streamErr)
	}

	if loadErrCh != nil {
		if loadErr := <-loadErrCh; loadErr != nil {
			buildErr = loadErr
			writeProgressEvent(progressWriter, imagetypes.ProgressEvent{
				Type:    "build",
				Service: serviceName,
				Error:   loadErr.Error(),
			})
			return nil, loadErr
		}
	}

	writeProgressEvent(progressWriter, imagetypes.ProgressEvent{
		Type:    "build",
		Phase:   "complete",
		Service: serviceName,
		Status:  "build complete",
	})

	digest := ""
	if resp != nil {
		if v, ok := resp.ExporterResponse["containerimage.digest"]; ok {
			digest = v
		}
	}

	return &imagetypes.BuildResult{
		Provider: providerName,
		Tags:     req.Tags,
		Digest:   digest,
	}, nil
}

func (s *BuildService) resolveProvider(override string) (string, arcane_build.BuildProvider, error) {
	providerName := strings.ToLower(strings.TrimSpace(override))
	if providerName == "" {
		if s.settings == nil {
			return "", nil, errors.New("settings service not available")
		}
		providerName = strings.ToLower(strings.TrimSpace(s.settings.GetSettingsConfig().BuildProvider.Value))
	}
	if providerName == "" {
		providerName = "local"
	}
	provider, ok := s.providers[providerName]
	if !ok {
		return "", nil, fmt.Errorf("unknown build provider: %s", providerName)
	}
	return providerName, provider, nil
}

func (s *BuildService) buildSolveOpt(ctx context.Context, req imagetypes.BuildRequest) (client.SolveOpt, <-chan error, error) {
	contextDir := filepath.Clean(req.ContextDir)

	dockerfilePath := strings.TrimSpace(req.Dockerfile)
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	fullDockerfilePath := dockerfilePath
	if !filepath.IsAbs(dockerfilePath) {
		fullDockerfilePath = filepath.Join(contextDir, dockerfilePath)
	}

	frontendAttrs := map[string]string{
		"filename": filepath.Base(fullDockerfilePath),
	}
	if strings.TrimSpace(req.Target) != "" {
		frontendAttrs["target"] = strings.TrimSpace(req.Target)
	}
	if len(req.Platforms) > 0 {
		frontendAttrs["platform"] = strings.Join(req.Platforms, ",")
	}
	for key, val := range req.BuildArgs {
		frontendAttrs[fmt.Sprintf("build-arg:%s", key)] = val
	}

	solveOpt := client.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: frontendAttrs,
		LocalDirs: map[string]string{
			"context":    contextDir,
			"dockerfile": filepath.Dir(fullDockerfilePath),
		},
	}

	var loadErrCh chan error
	exports := make([]client.ExportEntry, 0, 2)
	if req.Push {
		exports = append(exports, client.ExportEntry{
			Type: "image",
			Attrs: map[string]string{
				"name":           strings.Join(req.Tags, ","),
				"push":           "true",
				"oci-mediatypes": "true",
			},
		})
	}
	if req.Load {
		exportEntry, errCh, err := s.buildLoadExport(ctx, req.Tags)
		if err != nil {
			return client.SolveOpt{}, nil, err
		}
		loadErrCh = errCh
		exports = append(exports, exportEntry)
	}

	if len(exports) > 0 {
		solveOpt.Exports = exports
	}

	return solveOpt, loadErrCh, nil
}

func (s *BuildService) buildLoadExport(ctx context.Context, tags []string) (client.ExportEntry, chan error, error) {
	if s.dockerService == nil {
		return client.ExportEntry{}, nil, errors.New("docker service not available")
	}

	dockerClient, err := s.dockerService.GetClient()
	if err != nil {
		return client.ExportEntry{}, nil, fmt.Errorf("failed to connect to Docker: %w", err)
	}

	pr, pw := io.Pipe()
	loadErrCh := make(chan error, 1)
	go func() {
		defer pr.Close()
		_, loadErr := dockerClient.ImageLoad(ctx, pr)
		loadErrCh <- loadErr
	}()

	exportAttrs := map[string]string{}
	if len(tags) > 0 {
		exportAttrs["name"] = strings.Join(tags, ",")
	}

	return client.ExportEntry{
		Type:  "docker",
		Attrs: exportAttrs,
		Output: func(_ map[string]string) (io.WriteCloser, error) {
			return pw, nil
		},
	}, loadErrCh, nil
}

func normalizeBuildRequest(req imagetypes.BuildRequest, providerName string) imagetypes.BuildRequest {
	if !req.Push && !req.Load {
		if providerName == "depot" {
			req.Push = true
		} else {
			req.Load = true
		}
	}
	return req
}

func validateBuildRequest(req imagetypes.BuildRequest, providerName string) error {
	if strings.TrimSpace(req.ContextDir) == "" {
		return errors.New("contextDir is required")
	}

	contextDir := filepath.Clean(req.ContextDir)
	if _, err := os.Stat(contextDir); err != nil {
		return fmt.Errorf("build context not found: %w", err)
	}

	if providerName == "depot" && !req.Push {
		return errors.New("Depot builds must push images to a registry")
	}

	if len(req.Tags) == 0 && (req.Push || req.Load) {
		return errors.New("at least one tag is required when push/load is enabled")
	}

	dockerfilePath := strings.TrimSpace(req.Dockerfile)
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}
	fullDockerfilePath := dockerfilePath
	if !filepath.IsAbs(dockerfilePath) {
		fullDockerfilePath = filepath.Join(contextDir, dockerfilePath)
	}
	if _, err := os.Stat(fullDockerfilePath); err != nil {
		return fmt.Errorf("dockerfile not found: %w", err)
	}

	return nil
}

func normalizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		t := strings.TrimSpace(tag)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func streamSolveStatus(ctx context.Context, ch <-chan *client.SolveStatus, w io.Writer, serviceName string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case status, ok := <-ch:
			if !ok {
				return nil
			}
			if status == nil {
				continue
			}
			for _, s := range status.Statuses {
				if s == nil {
					continue
				}
				event := imagetypes.ProgressEvent{
					Type:    "build",
					Service: serviceName,
					ID:      s.ID,
					Status:  s.Name,
				}
				if s.Current > 0 || s.Total > 0 {
					event.ProgressDetail = &imagetypes.ProgressDetail{
						Current: s.Current,
						Total:   s.Total,
					}
				}
				writeProgressEvent(w, event)
			}
		}
	}
}

type flusher interface{ Flush() }

func writeProgressEvent(w io.Writer, event imagetypes.ProgressEvent) {
	if w == nil {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = w.Write(append(data, '\n'))
	if f, ok := w.(flusher); ok {
		f.Flush()
	}
}
