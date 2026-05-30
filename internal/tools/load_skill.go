package tools

import (
	"context"
	"encoding/json"

	"fmt"
	"strings"
	"vimo-chat/internal/skill"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type LoadSkillTool struct{}

type LoadSkillInput struct {
	SkillName string `json:"skill_name"`
}

// Info returns the tool information.
func (t *LoadSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "load_skill",
		Desc: `Load the full content of a skill into the agent's context.
		Use this when you need detailed information about how to handle a specific type of request.
		This will provide you with comprehensive instructions, policies, and guidelines for the skill area.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill_name": {
				Type:     schema.String,
				Desc:     "The name of the skill to load",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with the given input.
func (t *LoadSkillTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args LoadSkillInput

	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		// Fault-tolerant handling: If the model directly passes the plain text of the skill name
		args.SkillName = argumentsInJSON
	}

	for _, sk := range skill.Skills {
		if sk.Name == args.SkillName {
			return fmt.Sprintf("Loaded skill: %s\n\n%s", sk.Name, sk.Content), nil
		}
	}

	// Skill Not found
	var available []string
	for _, s := range skill.Skills {
		available = append(available, s.Name)
	}
	return fmt.Sprintf("Skill '%s' not found. Available skills: %s", args.SkillName, strings.Join(available, ", ")), nil
}

// Compile check
var _ tool.InvokableTool = &LoadSkillTool{}
