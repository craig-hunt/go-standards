package task

import "net/http"

const (
	PathTasks           = "/api/tasks"
	PathParamID         = "id"
	PathTask            = PathTasks + "/{" + PathParamID + "}"
	QueryFilter         = "filter"
	RouteList           = http.MethodGet + " " + PathTasks
	RouteCreate         = http.MethodPost + " " + PathTasks
	RouteClearCompleted = http.MethodDelete + " " + PathTasks
	RouteUpdate         = http.MethodPatch + " " + PathTask
	RouteDelete         = http.MethodDelete + " " + PathTask
)

const (
	FilterAll       Filter = "all"
	FilterActive    Filter = "active"
	FilterCompleted Filter = "completed"
)

const (
	MaxTitleLength = 200
	decimalBase    = 10
	idBits         = 64
	firstValidID   = 1
)

const (
	FieldTitle           = "title"
	FieldCompleted       = "completed"
	CodeInvalidFilter    = "invalid_filter"
	CodeInvalidID        = "invalid_id"
	CodeNotFound         = "not_found"
	MsgInvalidFilter     = "filter must be all, active, or completed"
	MsgInvalidID         = "task id must be a positive whole number"
	MsgNotFound          = "no task has that id"
	MsgTitleRequired     = "Enter a task title."
	MsgTitleTooLong      = "Keep the task title within the length limit."
	MsgCompletedRequired = "Say whether the task is completed."
)

const (
	errTitleRequired = "task title is required"
	errTitleTooLong  = "task title exceeds the length limit"
)

var SeedTitles = []Title{
	"Review the architecture decision record",
	"Reply to the vendor questionnaire",
}
