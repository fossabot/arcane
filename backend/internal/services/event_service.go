package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/utils/mapper"
	"github.com/getarcaneapp/arcane/backend/internal/utils/pagination"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/event"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type EventService struct {
	store database.EventStore
}

func NewEventService(store database.EventStore) *EventService {
	return &EventService{store: store}
}

type CreateEventRequest struct {
	Type          event.EventType     `json:"type"`
	Severity      event.EventSeverity `json:"severity,omitempty"`
	Title         string              `json:"title"`
	Description   string              `json:"description,omitempty"`
	ResourceType  *string             `json:"resourceType,omitempty"`
	ResourceID    *string             `json:"resourceId,omitempty"`
	ResourceName  *string             `json:"resourceName,omitempty"`
	UserID        *string             `json:"userId,omitempty"`
	Username      *string             `json:"username,omitempty"`
	EnvironmentID *string             `json:"environmentId,omitempty"`
	Metadata      base.JSON           `json:"metadata,omitempty"`
}

func (s *EventService) CreateEvent(ctx context.Context, req CreateEventRequest) (*event.ModelEvent, error) {
	severity := req.Severity
	if severity == "" {
		severity = event.EventSeverityInfo
	}

	now := time.Now()
	event := &event.ModelEvent{
		Type:          req.Type,
		Severity:      severity,
		Title:         req.Title,
		Description:   req.Description,
		ResourceType:  req.ResourceType,
		ResourceID:    req.ResourceID,
		ResourceName:  req.ResourceName,
		UserID:        req.UserID,
		Username:      req.Username,
		EnvironmentID: req.EnvironmentID,
		Metadata:      req.Metadata,
		Timestamp:     now,
		BaseModel: base.BaseModel{
			ID:        uuid.NewString(),
			CreatedAt: now,
			UpdatedAt: &now,
		},
	}

	created, err := s.store.CreateEvent(ctx, *event)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return created, nil
}

func (s *EventService) CreateEventFromDto(ctx context.Context, req event.CreateEvent) (*event.Event, error) {
	severity := event.EventSeverity(req.Severity)
	if severity == "" {
		severity = event.EventSeverityInfo
	}

	metadata := base.JSON{}
	if req.Metadata != nil {
		metadata = base.JSON(req.Metadata)
	}

	createReq := CreateEventRequest{
		Type:          event.EventType(req.Type),
		Severity:      severity,
		Title:         req.Title,
		Description:   req.Description,
		ResourceType:  req.ResourceType,
		ResourceID:    req.ResourceID,
		ResourceName:  req.ResourceName,
		UserID:        req.UserID,
		Username:      req.Username,
		EnvironmentID: req.EnvironmentID,
		Metadata:      metadata,
	}

	event, err := s.CreateEvent(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return s.toEventDto(event), nil
}

func (s *EventService) ListEventsPaginated(ctx context.Context, params pagination.QueryParams) ([]event.Event, pagination.Response, error) {
	return s.listEventsPaginatedInternal(ctx, params)
}

func (s *EventService) GetEventsByEnvironmentPaginated(ctx context.Context, environmentID string, params pagination.QueryParams) ([]event.Event, pagination.Response, error) {
	if params.Filters == nil {
		params.Filters = map[string]string{}
	}
	params.Filters["environmentId"] = environmentID
	return s.listEventsPaginatedInternal(ctx, params)
}

func (s *EventService) DeleteEvent(ctx context.Context, eventID string) error {
	deleted, err := s.store.DeleteEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	if !deleted {
		return fmt.Errorf("event not found")
	}
	return nil
}

func (s *EventService) DeleteOldEvents(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := s.store.DeleteEventsOlderThan(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old events: %w", err)
	}
	return nil
}

func (s *EventService) LogContainerEvent(ctx context.Context, eventType event.EventType, containerID, containerName, userID, username, environmentID string, metadata base.JSON) error {
	title := s.generateEventTitle(eventType, containerName)
	description := s.generateEventDescription(eventType, "container", containerName)
	severity := s.getEventSeverity(eventType)

	resourceType := "container"
	_, err := s.CreateEvent(ctx, CreateEventRequest{
		Type:          eventType,
		Severity:      severity,
		Title:         title,
		Description:   description,
		ResourceType:  &resourceType,
		ResourceID:    &containerID,
		ResourceName:  &containerName,
		UserID:        &userID,
		Username:      &username,
		EnvironmentID: &environmentID,
		Metadata:      metadata,
	})
	return err
}

func (s *EventService) LogImageEvent(ctx context.Context, eventType event.EventType, imageID, imageName, userID, username, environmentID string, metadata base.JSON) error {
	title := s.generateEventTitle(eventType, imageName)
	description := s.generateEventDescription(eventType, "image", imageName)
	severity := s.getEventSeverity(eventType)

	resourceType := "image"
	_, err := s.CreateEvent(ctx, CreateEventRequest{
		Type:          eventType,
		Severity:      severity,
		Title:         title,
		Description:   description,
		ResourceType:  &resourceType,
		ResourceID:    &imageID,
		ResourceName:  &imageName,
		UserID:        &userID,
		Username:      &username,
		EnvironmentID: &environmentID,
		Metadata:      metadata,
	})
	return err
}

func (s *EventService) LogProjectEvent(ctx context.Context, eventType event.EventType, projectID, projectName, userID, username, environmentID string, metadata base.JSON) error {
	title := s.generateEventTitle(eventType, projectName)
	description := s.generateEventDescription(eventType, "project", projectName)
	severity := s.getEventSeverity(eventType)

	resourceType := "project"
	_, err := s.CreateEvent(ctx, CreateEventRequest{
		Type:          eventType,
		Severity:      severity,
		Title:         title,
		Description:   description,
		ResourceType:  &resourceType,
		ResourceID:    &projectID,
		ResourceName:  &projectName,
		UserID:        &userID,
		Username:      &username,
		EnvironmentID: &environmentID,
		Metadata:      metadata,
	})
	return err
}

func (s *EventService) LogUserEvent(ctx context.Context, eventType event.EventType, userID, username string, metadata base.JSON) error {
	title := s.generateEventTitle(eventType, username)
	description := s.generateEventDescription(eventType, "user", username)
	severity := s.getEventSeverity(eventType)

	_, err := s.CreateEvent(ctx, CreateEventRequest{
		Type:        eventType,
		Severity:    severity,
		Title:       title,
		Description: description,
		UserID:      &userID,
		Username:    &username,
		Metadata:    metadata,
	})
	return err
}

func (s *EventService) LogVolumeEvent(ctx context.Context, eventType event.EventType, volumeID, volumeName, userID, username, environmentID string, metadata base.JSON) error {
	title := s.generateEventTitle(eventType, volumeName)
	description := s.generateEventDescription(eventType, "volume", volumeName)
	severity := s.getEventSeverity(eventType)

	resourceType := "volume"
	_, err := s.CreateEvent(ctx, CreateEventRequest{
		Type:          eventType,
		Severity:      severity,
		Title:         title,
		Description:   description,
		ResourceType:  &resourceType,
		ResourceID:    &volumeID,
		ResourceName:  &volumeName,
		UserID:        &userID,
		Username:      &username,
		EnvironmentID: &environmentID,
		Metadata:      metadata,
	})
	return err
}

func (s *EventService) LogNetworkEvent(ctx context.Context, eventType event.EventType, networkID, networkName, userID, username, environmentID string, metadata base.JSON) error {
	title := s.generateEventTitle(eventType, networkName)
	description := s.generateEventDescription(eventType, "network", networkName)
	severity := s.getEventSeverity(eventType)

	resourceType := "network"
	_, err := s.CreateEvent(ctx, CreateEventRequest{
		Type:          eventType,
		Severity:      severity,
		Title:         title,
		Description:   description,
		ResourceType:  &resourceType,
		ResourceID:    &networkID,
		ResourceName:  &networkName,
		UserID:        &userID,
		Username:      &username,
		EnvironmentID: &environmentID,
		Metadata:      metadata,
	})
	return err
}

func (s *EventService) LogErrorEvent(ctx context.Context, eventType event.EventType, resourceType, resourceID, resourceName, userID, username, environmentID string, err error, metadata base.JSON) {
	if err == nil {
		return
	}

	// Run error logging in background to prevent blocking the main flow
	// Detach context to ensure logging completes even if request is canceled
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		// Set a timeout for the background logging
		logCtx, cancel := context.WithTimeout(bgCtx, 30*time.Second)
		defer cancel()

		if metadata == nil {
			metadata = base.JSON{}
		}
		metadata["error"] = err.Error()

		titleCaser := cases.Title(language.English)
		title := fmt.Sprintf("%s error", titleCaser.String(resourceType))
		if resourceName != "" {
			title = fmt.Sprintf("%s error: %s", titleCaser.String(resourceType), resourceName)
		}

		description := fmt.Sprintf("Failed to perform operation on %s: %s", resourceType, err.Error())

		_, logErr := s.CreateEvent(logCtx, CreateEventRequest{
			Type:          eventType,
			Severity:      event.EventSeverityError,
			Title:         title,
			Description:   description,
			ResourceType:  &resourceType,
			ResourceID:    &resourceID,
			ResourceName:  &resourceName,
			UserID:        &userID,
			Username:      &username,
			EnvironmentID: &environmentID,
			Metadata:      metadata,
		})
		if logErr != nil {
			slog.ErrorContext(logCtx, "Failed to log error event", "error", logErr)
		}
	}()
}

func (s *EventService) listEventsPaginatedInternal(ctx context.Context, params pagination.QueryParams) ([]event.Event, pagination.Response, error) {
	if params.Limit != -1 {
		if params.Limit <= 0 {
			params.Limit = 20
		} else if params.Limit > 100 {
			params.Limit = 100
		}
	}
	if params.Start < 0 {
		params.Start = 0
	}

	events, err := s.store.ListEvents(ctx)
	if err != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to list events: %w", err)
	}

	config := pagination.Config[event.ModelEvent]{
		SearchAccessors: []pagination.SearchAccessor[event.ModelEvent]{
			func(e event.ModelEvent) (string, error) { return e.Title, nil },
			func(e event.ModelEvent) (string, error) { return e.Description, nil },
			func(e event.ModelEvent) (string, error) { return eventStringPtrValue(e.ResourceName), nil },
			func(e event.ModelEvent) (string, error) { return eventStringPtrValue(e.Username), nil },
		},
		SortBindings: []pagination.SortBinding[event.ModelEvent]{
			{Key: "type", Fn: func(a, b event.ModelEvent) int { return strings.Compare(string(a.Type), string(b.Type)) }},
			{Key: "severity", Fn: func(a, b event.ModelEvent) int { return strings.Compare(string(a.Severity), string(b.Severity)) }},
			{Key: "title", Fn: func(a, b event.ModelEvent) int { return strings.Compare(a.Title, b.Title) }},
			{Key: "resourceType", Fn: func(a, b event.ModelEvent) int {
				return strings.Compare(eventStringPtrValue(a.ResourceType), eventStringPtrValue(b.ResourceType))
			}},
			{Key: "resourceName", Fn: func(a, b event.ModelEvent) int {
				return strings.Compare(eventStringPtrValue(a.ResourceName), eventStringPtrValue(b.ResourceName))
			}},
			{Key: "username", Fn: func(a, b event.ModelEvent) int {
				return strings.Compare(eventStringPtrValue(a.Username), eventStringPtrValue(b.Username))
			}},
			{Key: "timestamp", Fn: func(a, b event.ModelEvent) int { return compareTime(a.Timestamp, b.Timestamp) }},
			{Key: "createdAt", Fn: func(a, b event.ModelEvent) int { return compareTime(a.CreatedAt, b.CreatedAt) }},
			{Key: "updatedAt", Fn: func(a, b event.ModelEvent) int { return compareOptionalTime(a.UpdatedAt, b.UpdatedAt) }},
		},
		FilterAccessors: []pagination.FilterAccessor[event.ModelEvent]{
			{
				Key: "severity",
				Fn: func(e event.ModelEvent, filterValue string) bool {
					return strings.EqualFold(strings.TrimSpace(string(e.Severity)), strings.TrimSpace(filterValue))
				},
			},
			{
				Key: "type",
				Fn: func(e event.ModelEvent, filterValue string) bool {
					return strings.EqualFold(strings.TrimSpace(string(e.Type)), strings.TrimSpace(filterValue))
				},
			},
			{
				Key: "resourceType",
				Fn: func(e event.ModelEvent, filterValue string) bool {
					return strings.EqualFold(strings.TrimSpace(eventStringPtrValue(e.ResourceType)), strings.TrimSpace(filterValue))
				},
			},
			{
				Key: "username",
				Fn: func(e event.ModelEvent, filterValue string) bool {
					return strings.EqualFold(strings.TrimSpace(eventStringPtrValue(e.Username)), strings.TrimSpace(filterValue))
				},
			},
			{
				Key: "environmentId",
				Fn: func(e event.ModelEvent, filterValue string) bool {
					return strings.EqualFold(strings.TrimSpace(eventStringPtrValue(e.EnvironmentID)), strings.TrimSpace(filterValue))
				},
			},
		},
	}

	result := pagination.SearchOrderAndPaginate(events, params, config)
	paginationResp := pagination.BuildResponseFromFilterResult(result, params)

	eventDtos, mapErr := mapper.MapSlice[event.ModelEvent, event.Event](result.Items)
	if mapErr != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to map events: %w", mapErr)
	}

	return eventDtos, paginationResp, nil
}

func eventStringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var eventDefinitions = map[event.EventType]struct {
	TitleFormat       string
	DescriptionFormat string
	Severity          event.EventSeverity
}{
	event.EventTypeContainerStart:   {"Container started: %s", "Container '%s' has been started", event.EventSeveritySuccess},
	event.EventTypeContainerStop:    {"Container stopped: %s", "Container '%s' has been stopped", event.EventSeverityInfo},
	event.EventTypeContainerRestart: {"Container restarted: %s", "Container '%s' has been restarted", event.EventSeverityInfo},
	event.EventTypeContainerDelete:  {"Container deleted: %s", "Container '%s' has been deleted", event.EventSeverityWarning},
	event.EventTypeContainerCreate:  {"Container created: %s", "Container '%s' has been created", event.EventSeveritySuccess},
	event.EventTypeContainerScan:    {"Container scanned: %s", "Security scan completed for container '%s'", event.EventSeverityInfo},
	event.EventTypeContainerUpdate:  {"Container updated: %s", "Container '%s' has been updated", event.EventSeverityInfo},
	event.EventTypeContainerError:   {"Container error: %s", "An error occurred with container '%s'", event.EventSeverityError},

	event.EventTypeImagePull:   {"Image pulled: %s", "Image '%s' has been pulled", event.EventSeveritySuccess},
	event.EventTypeImageLoad:   {"Image loaded: %s", "Image '%s' has been loaded from archive", event.EventSeveritySuccess},
	event.EventTypeImageDelete: {"Image deleted: %s", "Image '%s' has been deleted", event.EventSeverityWarning},
	event.EventTypeImageScan:   {"Image scanned: %s", "Security scan completed for image '%s'", event.EventSeverityInfo},
	event.EventTypeImageError:  {"Image error: %s", "An error occurred with image '%s'", event.EventSeverityError},

	event.EventTypeProjectDeploy: {"Project deployed: %s", "Project '%s' has been deployed", event.EventSeveritySuccess},
	event.EventTypeProjectDelete: {"Project deleted: %s", "Project '%s' has been deleted", event.EventSeverityWarning},
	event.EventTypeProjectStart:  {"Project started: %s", "Project '%s' has been started", event.EventSeveritySuccess},
	event.EventTypeProjectStop:   {"Project stopped: %s", "Project '%s' has been stopped", event.EventSeverityInfo},
	event.EventTypeProjectCreate: {"Project created: %s", "Project '%s' has been created", event.EventSeveritySuccess},
	event.EventTypeProjectUpdate: {"Project updated: %s", "Project '%s' has been updated", event.EventSeverityInfo},
	event.EventTypeProjectError:  {"Project error: %s", "An error occurred with project '%s'", event.EventSeverityError},

	event.EventTypeVolumeCreate:             {"Volume created: %s", "Volume '%s' has been created", event.EventSeveritySuccess},
	event.EventTypeVolumeDelete:             {"Volume deleted: %s", "Volume '%s' has been deleted", event.EventSeverityWarning},
	event.EventTypeVolumeError:              {"Volume error: %s", "An error occurred with volume '%s'", event.EventSeverityError},
	event.EventTypeVolumeFileCreate:         {"Volume file created: %s", "A file or directory was created in volume '%s'", event.EventSeveritySuccess},
	event.EventTypeVolumeFileDelete:         {"Volume file deleted: %s", "A file or directory was deleted in volume '%s'", event.EventSeverityWarning},
	event.EventTypeVolumeFileUpload:         {"Volume file uploaded: %s", "A file was uploaded to volume '%s'", event.EventSeveritySuccess},
	event.EventTypeVolumeBackupCreate:       {"Volume backup created: %s", "A backup was created for volume '%s'", event.EventSeveritySuccess},
	event.EventTypeVolumeBackupDelete:       {"Volume backup deleted: %s", "A backup was deleted for volume '%s'", event.EventSeverityWarning},
	event.EventTypeVolumeBackupRestore:      {"Volume backup restored: %s", "A backup was restored for volume '%s'", event.EventSeverityWarning},
	event.EventTypeVolumeBackupRestoreFiles: {"Volume backup files restored: %s", "Selected files were restored for volume '%s'", event.EventSeverityWarning},
	event.EventTypeVolumeBackupDownload:     {"Volume backup downloaded: %s", "A backup was downloaded for volume '%s'", event.EventSeverityInfo},

	event.EventTypeNetworkCreate: {"Network created: %s", "Network '%s' has been created", event.EventSeveritySuccess},
	event.EventTypeNetworkDelete: {"Network deleted: %s", "Network '%s' has been deleted", event.EventSeverityWarning},
	event.EventTypeNetworkError:  {"Network error: %s", "An error occurred with network '%s'", event.EventSeverityError},

	event.EventTypeSystemPrune:      {"System prune completed", "System resources have been pruned", event.EventSeverityInfo},
	event.EventTypeSystemAutoUpdate: {"System auto-update completed", "System auto-update process has completed", event.EventSeverityInfo},
	event.EventTypeSystemUpgrade:    {"System upgrade completed", "System upgrade process has completed", event.EventSeverityInfo},

	event.EventTypeUserLogin:  {"User logged in: %s", "User '%s' has logged in", event.EventSeverityInfo},
	event.EventTypeUserLogout: {"User logged out: %s", "User '%s' has logged out", event.EventSeverityInfo},
}

func (s *EventService) toEventDto(e *event.ModelEvent) *event.Event {
	var metadata map[string]interface{}
	if e.Metadata != nil {
		metadata = map[string]interface{}(e.Metadata)
	}

	return &event.Event{
		ID:            e.ID,
		Type:          string(e.Type),
		Severity:      string(e.Severity),
		Title:         e.Title,
		Description:   e.Description,
		ResourceType:  e.ResourceType,
		ResourceID:    e.ResourceID,
		ResourceName:  e.ResourceName,
		UserID:        e.UserID,
		Username:      e.Username,
		EnvironmentID: e.EnvironmentID,
		Metadata:      metadata,
		Timestamp:     e.Timestamp,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

func (s *EventService) generateEventTitle(eventType event.EventType, resourceName string) string {
	if def, ok := eventDefinitions[eventType]; ok {
		return fmt.Sprintf(def.TitleFormat, resourceName)
	}
	return fmt.Sprintf("Event: %s", string(eventType))
}

func (s *EventService) generateEventDescription(eventType event.EventType, resourceType, resourceName string) string {
	if def, ok := eventDefinitions[eventType]; ok {
		return fmt.Sprintf(def.DescriptionFormat, resourceName)
	}
	return fmt.Sprintf("%s operation performed on %s '%s'", string(eventType), resourceType, resourceName)
}

func (s *EventService) getEventSeverity(eventType event.EventType) event.EventSeverity {
	if def, ok := eventDefinitions[eventType]; ok {
		return def.Severity
	}
	return event.EventSeverityInfo
}
