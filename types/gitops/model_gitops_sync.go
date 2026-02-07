package gitops

import (
	"time"

	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/environment"
	"github.com/getarcaneapp/arcane/types/project"
)

// ModelGitOpsSync is the persisted GitOps sync model used by the backend data layer.
type ModelGitOpsSync struct {
	Name           string                        `json:"name" sortable:"true" search:"sync,gitops,automation,deploy,deployment,continuous"`
	EnvironmentID  string                        `json:"environmentId" sortable:"true"`
	Environment    *environment.ModelEnvironment `json:"environment,omitempty"`
	RepositoryID   string                        `json:"repositoryId" sortable:"true"`
	Repository     *ModelGitRepository           `json:"repository,omitempty"`
	Branch         string                        `json:"branch" sortable:"true" search:"branch,main,master,develop,feature,release"`
	ComposePath    string                        `json:"composePath" sortable:"true" search:"compose,docker-compose,path,file,yaml,yml"`
	ProjectName    string                        `json:"projectName" sortable:"true" search:"project,name,stack,application,service"`
	ProjectID      *string                       `json:"projectId,omitempty" sortable:"true"`
	Project        *project.Project              `json:"project,omitempty"`
	AutoSync       bool                          `json:"autoSync" sortable:"true" search:"auto,automatic,sync,continuous,scheduled"`
	SyncInterval   int                           `json:"syncInterval" sortable:"true" search:"interval,frequency,schedule,cron,minutes"`
	LastSyncAt     *time.Time                    `json:"lastSyncAt,omitempty" sortable:"true"`
	LastSyncStatus *string                       `json:"lastSyncStatus,omitempty" search:"status,success,failed,pending,error"`
	LastSyncError  *string                       `json:"lastSyncError,omitempty"`
	LastSyncCommit *string                       `json:"lastSyncCommit,omitempty" search:"commit,hash,sha,revision"`
	base.BaseModel
}
