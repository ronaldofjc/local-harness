package workflows

// PromptInfo representa informacoes de um prompt MCP derivado de um workflow.
type PromptInfo struct {
	Name        string
	Description string
}

// PromptMessageInfo representa uma mensagem de prompt.
type PromptMessageInfo struct {
	Role    string
	Content string
}

// ToPrompts converte workflows para informacoes de prompts.
func ToPrompts(workflows []Workflow) []PromptInfo {
	prompts := make([]PromptInfo, 0, len(workflows))
	for _, wf := range workflows {
		prompts = append(prompts, PromptInfo{
			Name:        "workflow." + wf.ID,
			Description: wf.Description,
		})
	}
	return prompts
}

// GetPromptMessages retorna as mensagens de um workflow.
func GetPromptMessages(wf *Workflow) []PromptMessageInfo {
	return []PromptMessageInfo{
		{
			Role:    "user",
			Content: wf.Content,
		},
	}
}
