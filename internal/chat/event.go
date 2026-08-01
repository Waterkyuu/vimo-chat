package chat

type EventType string

const (
	EventAssistantDelta EventType = "assistant_delta"
	EventToolStart      EventType = "tool_start"
	EventToolResult     EventType = "tool_result"
	EventError          EventType = "error"
	EventDone           EventType = "done"
)

type Event struct {
	Type       EventType
	Content    string
	ToolCallID string
	ToolName   string
	Arguments  string
	Err        error
}
