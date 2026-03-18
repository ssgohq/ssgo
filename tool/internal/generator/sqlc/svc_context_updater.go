// Package sqlc provides code generation for SQLC-based database layers
package sqlc

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// SvcContextUpdater updates ServiceContext using AST manipulation for SQLC-based services
type SvcContextUpdater struct {
	outputDir string
	module    string
	verbose   bool
}

// NewSvcContextUpdater creates a new SvcContextUpdater
func NewSvcContextUpdater(outputDir, module string, verbose bool) *SvcContextUpdater {
	return &SvcContextUpdater{
		outputDir: outputDir,
		module:    module,
		verbose:   verbose,
	}
}

// updateChanges holds the changes to be applied to ServiceContext
type updateChanges struct {
	needsStoreField bool
	needsStoreInit  bool
}

// isEmpty returns true if no changes are needed
func (c *updateChanges) isEmpty() bool {
	return !c.needsStoreField && !c.needsStoreInit
}

// Update updates ServiceContext to include Store
func (u *SvcContextUpdater) Update() error {
	path := filepath.Join(u.outputDir, "internal", "svc", "service_context.go")

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		u.logVerbose("  ServiceContext not found at %s, skipping update\n", path)
		return nil
	}

	// Read and parse file
	originalContent, file, err := u.readAndParseFile(path)
	if err != nil {
		return err
	}

	// Collect changes
	changes := u.collectChanges(file)
	if changes.isEmpty() {
		u.logVerbose("  ServiceContext already up to date\n")
		return nil
	}

	// Apply changes
	content := u.applyChanges(string(originalContent), changes)

	// Format and write
	if err := u.formatAndWrite(path, content); err != nil {
		return err
	}

	u.logChanges(changes)
	return nil
}

// logVerbose logs a message if verbose mode is enabled
func (u *SvcContextUpdater) logVerbose(format string, args ...interface{}) {
	if u.verbose {
		fmt.Printf(format, args...)
	}
}

// readAndParseFile reads and parses the ServiceContext file
func (u *SvcContextUpdater) readAndParseFile(path string) ([]byte, *ast.File, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read ServiceContext: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse ServiceContext: %w", err)
	}

	return content, file, nil
}

// collectChanges determines what changes need to be made
func (u *SvcContextUpdater) collectChanges(file *ast.File) *updateChanges {
	existingFields := u.findExistingFields(file)
	existingInits := u.findExistingInits(file)

	return &updateChanges{
		needsStoreField: !existingFields["Store"],
		needsStoreInit:  !existingInits["Store"],
	}
}

// applyChanges applies the collected changes to the content
func (u *SvcContextUpdater) applyChanges(content string, changes *updateChanges) string {
	if changes.needsStoreField {
		content = u.addImports(content)
	}

	if changes.needsStoreField {
		content = u.addStoreField(content)
	}

	if changes.needsStoreInit {
		content = u.updateNewServiceContext(content)
	}

	return content
}

// formatAndWrite formats the content and writes it to the file
func (u *SvcContextUpdater) formatAndWrite(path, content string) error {
	formatted, err := format.Source([]byte(content))
	if err != nil {
		u.logVerbose("  Warning: could not format ServiceContext: %v\n", err)
		formatted = []byte(content)
	}

	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return fmt.Errorf("failed to write ServiceContext: %w", err)
	}

	return nil
}

// logChanges logs the changes made
func (u *SvcContextUpdater) logChanges(changes *updateChanges) {
	if !u.verbose {
		return
	}

	var changeList []string
	if changes.needsStoreField {
		changeList = append(changeList, "Store field")
	}
	if changes.needsStoreInit {
		changeList = append(changeList, "Store init")
	}
	fmt.Printf("  Updated ServiceContext: added %s\n", strings.Join(changeList, ", "))
}

// findExistingFields finds existing fields in ServiceContext struct
func (u *SvcContextUpdater) findExistingFields(file *ast.File) map[string]bool {
	fields := make(map[string]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "ServiceContext" {
			return true
		}

		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}

		for _, field := range st.Fields.List {
			for _, name := range field.Names {
				fields[name.Name] = true
			}
		}
		return false
	})

	return fields
}

// findExistingInits finds existing initializations in NewServiceContext
func (u *SvcContextUpdater) findExistingInits(file *ast.File) map[string]bool {
	inits := make(map[string]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "NewServiceContext" {
			return true
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			// Check for svc.Store = ... pattern
			assign, ok := n.(*ast.AssignStmt)
			if ok {
				for _, lhs := range assign.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if ok {
						inits[sel.Sel.Name] = true
					}
				}
			}

			// Check for Store: ... pattern in composite literal
			kv, ok := n.(*ast.KeyValueExpr)
			if ok {
				if ident, ok := kv.Key.(*ast.Ident); ok {
					inits[ident.Name] = true
				}
			}

			return true
		})

		return false
	})

	return inits
}

// addImports adds store import to the file
func (u *SvcContextUpdater) addImports(content string) string {
	storeImport := fmt.Sprintf("\t\"%s/internal/store\"", u.module)

	// Check if import already exists
	if strings.Contains(content, storeImport) {
		return content
	}

	// Find import block
	importStart := strings.Index(content, "import (")
	if importStart == -1 {
		// Single import, need to convert to block
		singleImport := strings.Index(content, "import ")
		if singleImport == -1 {
			return content
		}
		// Find the end of single import line
		importEnd := strings.Index(content[singleImport:], "\n")
		if importEnd == -1 {
			return content
		}
		singleImportLine := content[singleImport : singleImport+importEnd]
		// Extract the import path
		importPath := strings.TrimPrefix(singleImportLine, "import ")
		// Convert to block
		newImportBlock := fmt.Sprintf("import (\n%s\n%s\n)", importPath, storeImport)
		return content[:singleImport] + newImportBlock + content[singleImport+importEnd:]
	}

	// Find end of import block
	importEnd := strings.Index(content[importStart:], ")")
	if importEnd == -1 {
		return content
	}
	importEnd += importStart

	// Check if there are internal imports already
	beforeClose := content[:importEnd]
	if strings.Contains(beforeClose, "/internal/") {
		// Insert after last internal import
		lastInternal := strings.LastIndex(beforeClose, "/internal/")
		insertPos := strings.Index(beforeClose[lastInternal:], "\n")
		if insertPos != -1 {
			insertPos += lastInternal + 1
			return content[:insertPos] + storeImport + "\n" + content[insertPos:]
		}
	}

	// Insert before closing paren with proper grouping
	return content[:importEnd] + "\n" + storeImport + "\n" + content[importEnd:]
}

// addStoreField adds Store field to ServiceContext struct
func (u *SvcContextUpdater) addStoreField(content string) string {
	// Find ServiceContext struct
	structStart := strings.Index(content, "type ServiceContext struct {")
	if structStart == -1 {
		return content
	}

	// Find where to insert (after Config field or at the start)
	structBodyStart := structStart + len("type ServiceContext struct {")

	// Look for Config field
	configPos := strings.Index(content[structBodyStart:], "Config")
	if configPos != -1 {
		// Find end of Config line
		lineEnd := strings.Index(content[structBodyStart+configPos:], "\n")
		if lineEnd != -1 {
			insertPos := structBodyStart + configPos + lineEnd + 1
			return content[:insertPos] + "\tStore  *store.Store\n" + content[insertPos:]
		}
	}

	// Insert at start of struct
	return content[:structBodyStart] + "\n\tStore *store.Store\n" + content[structBodyStart:]
}

// updateNewServiceContext updates the NewServiceContext function
func (u *SvcContextUpdater) updateNewServiceContext(content string) string {
	// Find NewServiceContext function
	funcStart := strings.Index(content, "func NewServiceContext(")
	if funcStart == -1 {
		return content
	}

	// Find function end
	funcEnd := u.findFuncEnd(content, funcStart)

	// Check which pattern is used
	funcBody := content[funcStart:funcEnd]

	if strings.Contains(funcBody, "return &ServiceContext{") {
		// Direct return pattern
		return u.updateDirectReturnPattern(content, funcStart, funcEnd)
	} else if strings.Contains(funcBody, "svc :=") || strings.Contains(funcBody, "svc =") {
		// Variable pattern
		return u.updateSvcVarPattern(content, funcStart, funcEnd)
	}

	return content
}

// findFuncEnd finds the end of a function
func (u *SvcContextUpdater) findFuncEnd(content string, funcStart int) int {
	braceCount := 0
	inFunc := false
	for i := funcStart; i < len(content); i++ {
		switch content[i] {
		case '{':
			braceCount++
			inFunc = true
		case '}':
			braceCount--
			if inFunc && braceCount == 0 {
				return i + 1
			}
		}
	}
	return len(content)
}

// updateDirectReturnPattern handles "return &ServiceContext{...}" pattern
func (u *SvcContextUpdater) updateDirectReturnPattern(content string, funcStart, funcEnd int) string {
	// Find the function signature
	sigEnd := strings.Index(content[funcStart:], "{")
	if sigEnd == -1 {
		return content
	}

	sig := content[funcStart : funcStart+sigEnd]
	returnsError := strings.Contains(sig, "error")

	// Find "return &ServiceContext{"
	returnPos := strings.Index(content[funcStart:funcEnd], "return &ServiceContext{")
	if returnPos == -1 {
		return content
	}
	absReturnPos := funcStart + returnPos

	// Find the composite literal content
	litStart := absReturnPos + len("return &ServiceContext{")
	litEnd := u.findMatchingBrace(content, litStart, funcEnd)

	// Build new function body
	existingFields := strings.TrimSpace(content[litStart:litEnd])
	newBody := u.buildDirectReturnBody(existingFields)

	// Find function body end
	funcBodyEnd := u.findFuncBodyEnd(content, absReturnPos, funcEnd)

	result := content[:absReturnPos] + newBody + content[funcBodyEnd:]

	// Update function signature to return error if not already
	if !returnsError {
		result = strings.Replace(result, "*ServiceContext {", "(*ServiceContext, error) {", 1)
	}

	return result
}

// findMatchingBrace finds the position of the matching closing brace
func (u *SvcContextUpdater) findMatchingBrace(content string, start, end int) int {
	braceCount := 1
	for i := start; i < end; i++ {
		switch content[i] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				return i
			}
		}
	}
	return end
}

// findFuncBodyEnd finds the end of the function body
func (u *SvcContextUpdater) findFuncBodyEnd(content string, absReturnPos, funcEnd int) int {
	funcBodyEnd := funcEnd - 1
	for funcBodyEnd > absReturnPos && content[funcBodyEnd] != '}' {
		funcBodyEnd--
	}
	return funcBodyEnd
}

// buildDirectReturnBody builds the new function body for direct return pattern
func (u *SvcContextUpdater) buildDirectReturnBody(existingFields string) string {
	var newBody strings.Builder

	newBody.WriteString(`	// Initialize database
	pool, err := store.NewPgxPool(&c.DB)
	if err != nil {
		return nil, err
	}
	st := store.NewStore(pool)

	svc := &ServiceContext{
`)

	if existingFields != "" {
		lines := strings.Split(existingFields, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "Store:") {
				continue
			}
			if !strings.HasSuffix(line, ",") {
				line += ","
			}
			newBody.WriteString("\t\t" + line + "\n")
		}
	}

	newBody.WriteString("\t\tStore: st,\n")
	newBody.WriteString("\t}\n")
	newBody.WriteString("\n\treturn svc, nil\n")

	return newBody.String()
}

// updateSvcVarPattern handles "svc := &ServiceContext{...}" pattern
func (u *SvcContextUpdater) updateSvcVarPattern(content string, funcStart, funcEnd int) string {
	// Find return statement to verify pattern
	if !strings.Contains(content[funcStart:funcEnd], "return svc") {
		return content
	}

	// Find svc assignment
	svcAssignPos := strings.Index(content[funcStart:funcEnd], "svc :=")
	if svcAssignPos == -1 {
		svcAssignPos = strings.Index(content[funcStart:funcEnd], "svc =")
	}

	if svcAssignPos != -1 {
		absSvcPos := funcStart + svcAssignPos

		// Insert Store initialization before svc assignment
		storeInit := `
	// Initialize database
	pool, err := store.NewPgxPool(&c.DB)
	if err != nil {
		return nil, err
	}
	st := store.NewStore(pool)

`
		content = content[:absSvcPos] + storeInit + content[absSvcPos:]
		offset := len(storeInit)

		// Add Store: st to svc composite literal
		litStart := strings.Index(content[absSvcPos+offset:], "{")
		if litStart != -1 {
			absLitStart := absSvcPos + offset + litStart + 1
			content = content[:absLitStart] + "\n\t\tStore: st," + content[absLitStart:]
		}
	}

	return content
}

// UpdateMainGo updates cmd/main.go to handle error from NewServiceContext
func (u *SvcContextUpdater) UpdateMainGo() error {
	// Try common main.go locations
	mainPaths := []string{
		filepath.Join(u.outputDir, "cmd", "main.go"),
		filepath.Join(u.outputDir, "main.go"),
	}

	var mainPath string
	for _, p := range mainPaths {
		if _, err := os.Stat(p); err == nil {
			mainPath = p
			break
		}
	}

	if mainPath == "" {
		u.logVerbose("  main.go not found, skipping update\n")
		return nil
	}

	content, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("failed to read main.go: %w", err)
	}

	contentStr := string(content)

	// Check if already updated (error handling exists)
	if strings.Contains(contentStr, "svcCtx, err :=") ||
		strings.Contains(contentStr, "svcCtx, err=") {
		u.logVerbose("  main.go already handles NewServiceContext error\n")
		return nil
	}

	// Find and replace the simple assignment pattern
	// Pattern: svcCtx := svc.NewServiceContext(cfg)
	oldPattern := "svcCtx := svc.NewServiceContext(cfg)"
	if !strings.Contains(contentStr, oldPattern) {
		// Try alternate pattern with config variable name 'c'
		oldPattern = "svcCtx := svc.NewServiceContext(c)"
	}
	if !strings.Contains(contentStr, oldPattern) {
		u.logVerbose("  Could not find NewServiceContext call in main.go\n")
		return nil
	}

	// Determine the config variable name
	configVar := "cfg"
	if strings.Contains(oldPattern, "(c)") {
		configVar = "c"
	}

	newPattern := fmt.Sprintf(`svcCtx, err := svc.NewServiceContext(%s)
	if err != nil {
		panic(fmt.Sprintf("failed to create service context: %%v", err))
	}`, configVar)

	contentStr = strings.Replace(contentStr, oldPattern, newPattern, 1)

	formatted, err := format.Source([]byte(contentStr))
	if err != nil {
		u.logVerbose("  Warning: could not format main.go: %v\n", err)
		formatted = []byte(contentStr)
	}

	cleanMainPath := filepath.Clean(mainPath)
	if err := os.WriteFile(cleanMainPath, formatted, 0o644); err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}

	u.logVerbose("  Updated main.go to handle NewServiceContext error\n")
	return nil
}
