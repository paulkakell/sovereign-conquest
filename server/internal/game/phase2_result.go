package game

type phase2Result struct {
	OK        bool
	Message   string
	ErrorCode string
	Logs      []logToInsert
}
