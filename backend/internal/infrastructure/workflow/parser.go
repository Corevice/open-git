package workflow

import (
	"fmt"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

var yamlLineRegexp = regexp.MustCompile(`line (\d+)`)

type Step struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
	Uses string `yaml:"uses"`
}

type Job struct {
	Steps []Step `yaml:"steps"`
}

type Workflow struct {
	Name string         `yaml:"name"`
	On   yaml.Node      `yaml:"on"`
	Jobs map[string]Job `yaml:"jobs"`
}

type ParseError struct {
	Line    int
	Message string
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("workflow parse error at line %d: %s", e.Line, e.Message)
	}
	return fmt.Sprintf("workflow parse error: %s", e.Message)
}

func ParseWorkflow(data []byte) (*Workflow, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		line := extractLineFromMessage(err.Error())
		if typeErr, ok := err.(*yaml.TypeError); ok && len(typeErr.Errors) > 0 && line == 0 {
			line = extractLineFromMessage(typeErr.Errors[0])
		}
		if line == 0 {
			line = 1
		}
		return nil, &ParseError{Line: line, Message: err.Error()}
	}

	if root.Kind == 0 || (root.Kind == yaml.DocumentNode && len(root.Content) == 0) {
		return nil, &ParseError{Line: 1, Message: "empty workflow document"}
	}

	doc := &root
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) == 0 {
			return nil, &ParseError{Line: 1, Message: "empty workflow document"}
		}
		doc = root.Content[0]
	}
	if doc.Kind != yaml.MappingNode {
		return nil, &ParseError{Line: doc.Line, Message: "workflow root must be a mapping"}
	}

	var wf Workflow
	if err := doc.Decode(&wf); err != nil {
		return nil, &ParseError{Line: doc.Line, Message: err.Error()}
	}

	if len(wf.Jobs) == 0 {
		return nil, &ParseError{Line: doc.Line, Message: "workflow must define at least one job"}
	}
	for jobID, job := range wf.Jobs {
		if len(job.Steps) == 0 {
			return nil, &ParseError{Line: doc.Line, Message: fmt.Sprintf("job %q must define at least one step", jobID)}
		}
	}

	return &wf, nil
}

func extractLineFromMessage(msg string) int {
	m := yamlLineRegexp.FindStringSubmatch(msg)
	if len(m) < 2 {
		return 0
	}
	line, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return line
}
