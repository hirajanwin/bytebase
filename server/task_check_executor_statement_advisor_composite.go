package server

import (
	"context"
	"encoding/json"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/store"
)

// Schema review policy consists of a list of schema review rules.
// There is such a logical mapping in Bytebase backend:
//   1. One schema review policy maps a TaskCheckRun.
//   2. Each schema review rule type maps an advisor.Type.
//   3. Each [db.Type][AdvisorType] maps an advisor.

// NewTaskCheckStatementAdvisorCompositeExecutor creates a task check statement advisor composite executor.
func NewTaskCheckStatementAdvisorCompositeExecutor() TaskCheckExecutor {
	return &TaskCheckStatementAdvisorCompositeExecutor{}
}

// TaskCheckStatementAdvisorCompositeExecutor is the task check statement advisor composite executor with has sub-advisor.
type TaskCheckStatementAdvisorCompositeExecutor struct {
}

// Run will run the task check statement advisor composite executor once, and run its sub-advisor one-by-one.
func (*TaskCheckStatementAdvisorCompositeExecutor) Run(ctx context.Context, server *Server, taskCheckRun *api.TaskCheckRun) (result []api.TaskCheckResult, err error) {
	if taskCheckRun.Type != api.TaskCheckDatabaseStatementAdvise {
		return nil, common.Errorf(common.Invalid, "invalid check statement advisor composite type: %v", taskCheckRun.Type)
	}
	if !server.feature(api.FeatureSQLReviewPolicy) {
		return nil, common.Errorf(common.NotAuthorized, api.FeatureSQLReviewPolicy.AccessErrorMessage())
	}

	payload := &api.TaskCheckDatabaseStatementAdvisePayload{}
	if err := json.Unmarshal([]byte(taskCheckRun.Payload), payload); err != nil {
		return nil, common.Errorf(common.Invalid, "invalid check statement advise payload: %w", err)
	}

	policy, err := server.store.GetNormalSchemaReviewPolicy(ctx, &api.PolicyFind{ID: &payload.PolicyID})
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			return []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusWarn,
					Namespace: api.AdvisorNamespace,
					Code:      advisor.NotFound.Int(),
					Title:     "Empty schema review policy or disabled",
					Content:   "",
				},
			}, nil
		}
		return nil, common.Errorf(common.Internal, "failed to get schema review policy: %w", err)
	}

	task, err := server.store.GetTaskByID(ctx, taskCheckRun.TaskID)
	if err != nil {
		return nil, common.Errorf(common.Internal, "failed to get task by id: %w", err)
	}

	catalog := store.NewCatalog(task.DatabaseID, server.store, payload.DbType)

	dbType, err := api.ConvertToAdvisorDBType(payload.DbType)
	if err != nil {
		return nil, err
	}

	adviceList, err := advisor.SchemaReviewCheck(payload.Statement, policy.RuleList, advisor.SQLReviewCheckContext{
		Charset:   payload.Charset,
		Collation: payload.Collation,
		DbType:    dbType,
		Catalog:   catalog,
	})
	if err != nil {
		return nil, err
	}

	result = []api.TaskCheckResult{}
	for _, advice := range adviceList {
		status := api.TaskCheckStatusSuccess
		switch advice.Status {
		case advisor.Success:
			continue
		case advisor.Warn:
			status = api.TaskCheckStatusWarn
		case advisor.Error:
			status = api.TaskCheckStatusError
		}

		result = append(result, api.TaskCheckResult{
			Status:    status,
			Namespace: api.AdvisorNamespace,
			Code:      advice.Code.Int(),
			Title:     advice.Title,
			Content:   advice.Content,
		})
	}

	if len(result) == 0 {
		result = append(result, api.TaskCheckResult{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   "",
		})
	}

	return result, nil
}
