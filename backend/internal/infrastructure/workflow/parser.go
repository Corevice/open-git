package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxWorkflowSize = 512 * 1024
const maxMatrixCombinations = 256

var yamlLineRegexp = regexp.MustCompile(`line (\d+)`)

type Step struct {
	ID              string            `yaml:"id"`
	Name            string            `yaml:"name"`
	Uses            string            `yaml:"uses"`
	Run             string            `yaml:"run"`
	With            map[string]string `yaml:"with"`
	Env             map[string]string `yaml:"env"`
	If              string            `yaml:"if"`
	ContinueOnError bool              `yaml:"continue-on-error"`
	WorkingDir      string            `yaml:"working-directory"`
	Shell           string            `yaml:"shell"`
}

type Job struct {
	RunsOn         yaml.Node         `yaml:"runs-on"`
	Needs          yaml.Node         `yaml:"needs"`
	If             string            `yaml:"if"`
	Steps          []Step            `yaml:"steps"`
	Strategy       StrategyConfig    `yaml:"strategy"`
	Env            map[string]string `yaml:"env"`
	Outputs        map[string]string `yaml:"outputs"`
	TimeoutMinutes int               `yaml:"timeout-minutes"`
	Container      string            `yaml:"container"`
	Services       map[string]string `yaml:"services"`
}

type StrategyConfig struct {
	Matrix      yaml.Node `yaml:"matrix"`
	FailFast    *bool     `yaml:"fail-fast"`
	MaxParallel int       `yaml:"max-parallel"`
}

type Diagnostic struct {
	Line     int
	Col      int
	Severity string
	Message  string
}

type UsesRef struct {
	Kind      string
	Owner     string
	Name      string
	Ref       string
	LocalPath string
	Image     string
}

type WorkflowIR struct {
	On   map[string]any
	Env  map[string]string
	Jobs map[string]IRJob
	DAG  DAGInfo
}

type IRJob struct {
	RunsOn          string
	Needs           []string
	MatrixExpansion []map[string]any
	Steps           []IRStep
	Env             map[string]string
}

type IRStep struct {
	ID      string
	Uses    string
	UsesRef *UsesRef
	Run     string
	Env     map[string]string
	With    map[string]string
	If      string
	IfAST   []ExprNode
}

type DAGInfo struct {
	Order []string
	Edges [][2]string
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

	doc := documentContent(&root)
	if doc == nil {
		return nil, &ParseError{Line: 1, Message: "empty workflow document"}
	}
	if doc.Kind != yaml.MappingNode {
		return nil, &ParseError{Line: doc.Line, Message: "workflow root must be a mapping"}
	}

	_, diags, _ := ParseWorkflowFull(data)
	for _, d := range diags {
		if d.Severity == "error" {
			return nil, &ParseError{Line: d.Line, Message: d.Message}
		}
	}

	var wf Workflow
	if err := doc.Decode(&wf); err != nil {
		return nil, &ParseError{Line: doc.Line, Message: err.Error()}
	}

	for jobID, job := range wf.Jobs {
		if len(job.Steps) == 0 {
			return nil, &ParseError{Line: doc.Line, Message: fmt.Sprintf("job %q must define at least one step", jobID)}
		}
	}

	return &wf, nil
}

func ParseWorkflowFull(data []byte) (*WorkflowIR, []Diagnostic, error) {
	var diags []Diagnostic

	if len(data) > maxWorkflowSize {
		diags = append(diags, Diagnostic{
			Line:     1,
			Col:      1,
			Severity: "error",
			Message:  fmt.Sprintf("workflow file exceeds maximum size of %d bytes", maxWorkflowSize),
		})
		return nil, diags, nil
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		line := extractLineFromMessage(err.Error())
		if line == 0 {
			line = 1
		}
		diags = append(diags, Diagnostic{Line: line, Col: 1, Severity: "error", Message: err.Error()})
		return nil, diags, nil
	}

	doc := documentContent(&root)
	if doc == nil {
		diags = append(diags, Diagnostic{Line: 1, Col: 1, Severity: "error", Message: "empty workflow document"})
		return nil, diags, nil
	}
	if doc.Kind != yaml.MappingNode {
		diags = append(diags, Diagnostic{Line: doc.Line, Col: doc.Col, Severity: "error", Message: "workflow root must be a mapping"})
		return nil, diags, nil
	}

	var onNode *yaml.Node
	var env map[string]string
	var jobsNode *yaml.Node
	for i := 0; i < len(doc.Content)-1; i += 2 {
		key := doc.Content[i].Value
		val := doc.Content[i+1]
		switch key {
		case "on":
			onNode = val
		case "env":
			_ = val.Decode(&env)
		case "jobs":
			jobsNode = val
		}
	}

	if onNode == nil {
		diags = append(diags, Diagnostic{Line: doc.Line, Col: doc.Col, Severity: "error", Message: "workflow must define trigger 'on'"})
	}
	if jobsNode == nil {
		diags = append(diags, Diagnostic{Line: doc.Line, Col: doc.Col, Severity: "error", Message: "workflow must contain at least one job"})
		return nil, diags, nil
	}

	var on map[string]any
	if onNode != nil {
		on = normalizeOnTrigger(*onNode)
	}

	var rawJobs map[string]Job
	if err := jobsNode.Decode(&rawJobs); err != nil {
		diags = append(diags, Diagnostic{Line: jobsNode.Line, Col: jobsNode.Col, Severity: "error", Message: err.Error()})
		return nil, diags, nil
	}
	if len(rawJobs) == 0 {
		diags = append(diags, Diagnostic{Line: jobsNode.Line, Col: jobsNode.Col, Severity: "error", Message: "workflow must contain at least one job"})
		return nil, diags, nil
	}

	ir := &WorkflowIR{
		On:   on,
		Env:  env,
		Jobs: make(map[string]IRJob, len(rawJobs)),
	}

	stepLines := extractStepLines(jobsNode)

	for jobID, job := range rawJobs {
		irJob := IRJob{
			RunsOn: nodeToString(job.RunsOn),
			Needs:  decodeNeeds(job.Needs),
			Env:    job.Env,
		}

		if job.Strategy.Matrix.Kind != 0 {
			expansion, matrixDiags := expandMatrix(job.Strategy.Matrix)
			diags = append(diags, matrixDiags...)
			irJob.MatrixExpansion = expansion
		}

		for i, step := range job.Steps {
			stepLine := jobsNode.Line
			if lines, ok := stepLines[jobID]; ok && i < len(lines) {
				stepLine = lines[i]
			}
			if step.Uses != "" && step.Run != "" {
				diags = append(diags, Diagnostic{
					Line:     stepLine,
					Col:      1,
					Severity: "error",
					Message:  "step cannot define both 'uses' and 'run'",
				})
			}

			irStep := IRStep{
				ID:   step.ID,
				Uses: step.Uses,
				Run:  step.Run,
				Env:  step.Env,
				With: step.With,
				If:   step.If,
			}

			if step.Uses != "" {
				ref, useDiag := resolveUses(step.Uses, stepLine)
				if useDiag != nil {
					diags = append(diags, *useDiag)
				}
				irStep.UsesRef = &ref
			}

			if step.If != "" {
				ifAST, ifDiags := ExtractExpressions(step.If, stepLine)
				if len(ifAST) == 0 && !strings.Contains(step.If, "${{") {
					parsed, parseDiags := ParseExpression(strings.TrimSpace(step.If))
					for j := range parseDiags {
						if parseDiags[j].Line > 0 {
							parseDiags[j].Line += stepLine - 1
						}
					}
					ifDiags = append(ifDiags, parseDiags...)
					ifAST = parsed
				}
				diags = append(diags, ifDiags...)
				irStep.IfAST = ifAST
			}
			if step.Run != "" {
				runAST, runDiags := ExtractExpressions(step.Run, stepLine)
				diags = append(diags, runDiags...)
				irStep.IfAST = append(irStep.IfAST, runAST...)
			}

			irJob.Steps = append(irJob.Steps, irStep)
		}

		if job.If != "" {
			_, ifDiags := ExtractExpressions(job.If, jobsNode.Line)
			diags = append(diags, ifDiags...)
		}

		ir.Jobs[jobID] = irJob
	}

	dag, dagDiags := buildDAG(ir.Jobs)
	diags = append(diags, dagDiags...)
	ir.DAG = dag

	return ir, diags, nil
}

func normalizeOnTrigger(node yaml.Node) map[string]any {
	result := make(map[string]any)
	switch node.Kind {
	case yaml.ScalarNode:
		result[node.Value] = map[string]any{}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			result[item.Value] = map[string]any{}
		}
	case yaml.MappingNode:
		_ = node.Decode(&result)
	}
	return result
}

func resolveUses(s string, line int) (UsesRef, *Diagnostic) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "docker://") {
		return UsesRef{Kind: "docker", Image: strings.TrimPrefix(s, "docker://")}, nil
	}
	if strings.HasPrefix(s, "./") {
		return UsesRef{Kind: "local", LocalPath: s}, nil
	}
	if strings.Contains(s, "@") {
		idx := strings.LastIndex(s, "@")
		action := s[:idx]
		ref := s[idx+1:]
		parts := strings.SplitN(action, "/", 2)
		ur := UsesRef{Kind: "remote", Ref: ref}
		if len(parts) == 2 {
			ur.Owner = parts[0]
			ur.Name = parts[1]
		} else {
			ur.Name = action
		}
		return ur, nil
	}
	return UsesRef{Kind: "remote"}, &Diagnostic{
		Line:     line,
		Col:      1,
		Severity: "error",
		Message:  "remote action reference missing @ref",
	}
}

func expandMatrix(node yaml.Node) ([]map[string]any, []Diagnostic) {
	var diags []Diagnostic
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		diags = append(diags, Diagnostic{Line: node.Line, Col: node.Col, Severity: "error", Message: err.Error()})
		return nil, diags
	}

	var include []map[string]any
	var exclude []map[string]any
	axes := make(map[string][]any)

	for key, val := range raw {
		switch key {
		case "include":
			include = decodeMatrixRows(val, node.Line, &diags)
		case "exclude":
			exclude = decodeMatrixRows(val, node.Line, &diags)
		default:
			if arr, ok := val.([]any); ok {
				axes[key] = arr
			} else {
				axes[key] = []any{val}
			}
		}
	}

	combos := cartesianProduct(axes)
	combos = applyInclude(combos, include)
	combos = applyExclude(combos, exclude)

	if len(combos) > maxMatrixCombinations {
		diags = append(diags, Diagnostic{
			Line:     node.Line,
			Col:      node.Col,
			Severity: "error",
			Message:  fmt.Sprintf("matrix expansion exceeds maximum of %d combinations (got %d)", maxMatrixCombinations, len(combos)),
		})
		return combos[:maxMatrixCombinations], diags
	}

	return combos, diags
}

func decodeMatrixRows(val any, line int, diags *[]Diagnostic) []map[string]any {
	items, ok := val.([]any)
	if !ok {
		*diags = append(*diags, Diagnostic{Line: line, Col: 1, Severity: "error", Message: "matrix include/exclude must be a sequence"})
		return nil
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			*diags = append(*diags, Diagnostic{Line: line, Col: 1, Severity: "error", Message: "matrix include/exclude row must be a mapping"})
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func cartesianProduct(axes map[string][]any) []map[string]any {
	keys := make([]string, 0, len(axes))
	for k := range axes {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return []map[string]any{{}}
	}

	result := []map[string]any{{}}
	for _, key := range keys {
		values := axes[key]
		var next []map[string]any
		for _, combo := range result {
			for _, val := range values {
				newCombo := make(map[string]any, len(combo)+1)
				for k, v := range combo {
					newCombo[k] = v
				}
				newCombo[key] = val
				next = append(next, newCombo)
			}
		}
		result = next
	}
	return result
}

func applyInclude(combos []map[string]any, include []map[string]any) []map[string]any {
	if len(include) == 0 {
		return combos
	}
	result := make([]map[string]any, len(combos))
	copy(result, combos)
	for _, row := range include {
		merged := make(map[string]any)
		if len(combos) > 0 {
			for k, v := range combos[0] {
				merged[k] = v
			}
		}
		for k, v := range row {
			merged[k] = v
		}
		result = append(result, merged)
	}
	return result
}

func applyExclude(combos []map[string]any, exclude []map[string]any) []map[string]any {
	if len(exclude) == 0 {
		return combos
	}
	var result []map[string]any
outer:
	for _, combo := range combos {
		for _, ex := range exclude {
			if rowMatches(combo, ex) {
				continue outer
			}
		}
		result = append(result, combo)
	}
	return result
}

func rowMatches(combo, pattern map[string]any) bool {
	for k, v := range pattern {
		if combo[k] != v {
			return false
		}
	}
	return true
}

func buildDAG(jobs map[string]IRJob) (DAGInfo, []Diagnostic) {
	var diags []Diagnostic
	inDegree := make(map[string]int, len(jobs))
	adj := make(map[string][]string, len(jobs))
	var edges [][2]string

	for jobID := range jobs {
		inDegree[jobID] = 0
	}

	for jobID, job := range jobs {
		for _, need := range job.Needs {
			if _, ok := jobs[need]; !ok {
				diags = append(diags, Diagnostic{
					Line:     1,
					Col:      1,
					Severity: "error",
					Message:  fmt.Sprintf("job %q needs unknown job %q", jobID, need),
				})
				continue
			}
			adj[need] = append(adj[need], jobID)
			inDegree[jobID]++
			edges = append(edges, [2]string{need, jobID})
		}
	}

	queue := make([]string, 0, len(jobs))
	for jobID, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, jobID)
		}
	}

	var order []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		order = append(order, cur)
		for _, next := range adj[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(order) < len(jobs) {
		cycleMsg := detectCycleMessage(jobs)
		diags = append(diags, Diagnostic{
			Line:     1,
			Col:      1,
			Severity: "error",
			Message:  cycleMsg,
		})
	}

	return DAGInfo{Order: order, Edges: edges}, diags
}

func detectCycleMessage(jobs map[string]IRJob) string {
	visited := make(map[string]int)
	var stack []string
	var cyclePath []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = 1
		stack = append(stack, node)
		for _, need := range jobs[node].Needs {
			if _, ok := jobs[need]; !ok {
				continue
			}
			if visited[need] == 1 {
				cyclePath = append([]string{}, stack...)
				cyclePath = append(cyclePath, need)
				return true
			}
			if visited[need] == 0 && dfs(need) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		visited[node] = 2
		return false
	}

	for jobID := range jobs {
		if visited[jobID] == 0 && dfs(jobID) && len(cyclePath) >= 2 {
			a := cyclePath[0]
			b := cyclePath[1]
			return fmt.Sprintf("cyclic dependency detected: %s → %s → %s", a, b, a)
		}
	}

	return "cyclic dependency detected"
}

func extractStepLines(jobsNode *yaml.Node) map[string][]int {
	result := make(map[string][]int)
	if jobsNode == nil || jobsNode.Kind != yaml.MappingNode {
		return result
	}
	for i := 0; i < len(jobsNode.Content)-1; i += 2 {
		jobID := jobsNode.Content[i].Value
		jobVal := jobsNode.Content[i+1]
		if jobVal.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j < len(jobVal.Content)-1; j += 2 {
			if jobVal.Content[j].Value != "steps" {
				continue
			}
			stepsNode := jobVal.Content[j+1]
			if stepsNode.Kind != yaml.SequenceNode {
				continue
			}
			for _, stepNode := range stepsNode.Content {
				result[jobID] = append(result[jobID], stepNode.Line)
			}
		}
	}
	return result
}

func decodeNeeds(node yaml.Node) []string {
	if node.Kind == 0 {
		return nil
	}
	if node.Kind == yaml.ScalarNode {
		return []string{node.Value}
	}
	var needs []string
	_ = node.Decode(&needs)
	return needs
}

func nodeToString(node yaml.Node) string {
	if node.Kind == 0 {
		return ""
	}
	if node.Kind == yaml.ScalarNode {
		return node.Value
	}
	if node.Kind == yaml.SequenceNode {
		parts := make([]string, 0, len(node.Content))
		for _, n := range node.Content {
			parts = append(parts, n.Value)
		}
		return strings.Join(parts, ",")
	}
	return ""
}

func documentContent(root *yaml.Node) *yaml.Node {
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) == 0 {
			return nil
		}
		return root.Content[0]
	}
	return root
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
