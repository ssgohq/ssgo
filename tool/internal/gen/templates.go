package gen

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"sync"
	"text/template"
)

// TemplateManager handles template loading, caching, and execution.
type TemplateManager struct {
	fsys    fs.FS
	baseDir string
	funcMap template.FuncMap

	mu          sync.RWMutex
	fileCache   map[string]*template.Template
	inlineCache map[string]*template.Template
}

// NewTemplateManager creates a new TemplateManager.
func NewTemplateManager(fsys fs.FS, baseDir string, funcMap template.FuncMap) *TemplateManager {
	return &TemplateManager{
		fsys:        fsys,
		baseDir:     baseDir,
		funcMap:     funcMap,
		fileCache:   make(map[string]*template.Template),
		inlineCache: make(map[string]*template.Template),
	}
}

// FuncMap returns the template function map.
func (m *TemplateManager) FuncMap() template.FuncMap {
	return m.funcMap
}

// Render renders a template file with the given data and returns the result.
func (m *TemplateManager) Render(name string, data any) (string, error) {
	tpl, err := m.getTemplate(name)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// RenderString renders an inline template string (cached).
// This is more efficient than parsing the same inline template repeatedly.
func (m *TemplateManager) RenderString(tplText string, data any) (string, error) {
	if tplText == "" {
		return "", nil
	}

	tpl, err := m.getInlineTemplate(tplText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute inline template: %w", err)
	}

	return buf.String(), nil
}

// RenderToFile renders a template and writes to file.
func (m *TemplateManager) RenderToFile(fm *FileManager, tplName, filePath string, data any) error {
	content, err := m.Render(tplName, data)
	if err != nil {
		return err
	}
	return fm.WriteFile(filePath, []byte(content))
}

// RenderSkipExisting renders and writes only if file doesn't exist.
// Returns skipped=true if file already exists.
func (m *TemplateManager) RenderSkipExisting(
	fm *FileManager,
	tplName, filePath string,
	data any,
) (skipped bool, err error) {
	content, err := m.Render(tplName, data)
	if err != nil {
		return false, err
	}
	return fm.WriteFileSkipExisting(filePath, []byte(content))
}

// getTemplate returns a cached template or loads it from file.
func (m *TemplateManager) getTemplate(name string) (*template.Template, error) {
	fullPath := path.Join(m.baseDir, name)

	// Check cache first
	m.mu.RLock()
	if tpl, ok := m.fileCache[fullPath]; ok {
		m.mu.RUnlock()
		return tpl, nil
	}
	m.mu.RUnlock()

	// Load template
	content, err := fs.ReadFile(m.fsys, fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", name, err)
	}

	tpl, err := template.New(fullPath).Funcs(m.funcMap).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	// Cache the template
	m.mu.Lock()
	m.fileCache[fullPath] = tpl
	m.mu.Unlock()

	return tpl, nil
}

// getInlineTemplate returns a cached inline template or parses it.
func (m *TemplateManager) getInlineTemplate(tplText string) (*template.Template, error) {
	// Check cache first
	m.mu.RLock()
	if tpl, ok := m.inlineCache[tplText]; ok {
		m.mu.RUnlock()
		return tpl, nil
	}
	m.mu.RUnlock()

	// Parse template
	tpl, err := template.New("inline").Funcs(m.funcMap).Parse(tplText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse inline template: %w", err)
	}

	// Cache the template
	m.mu.Lock()
	m.inlineCache[tplText] = tpl
	m.mu.Unlock()

	return tpl, nil
}

// ClearCache clears both file and inline template caches.
func (m *TemplateManager) ClearCache() {
	m.mu.Lock()
	m.fileCache = make(map[string]*template.Template)
	m.inlineCache = make(map[string]*template.Template)
	m.mu.Unlock()
}

// ClearFileCache clears only the file template cache.
func (m *TemplateManager) ClearFileCache() {
	m.mu.Lock()
	m.fileCache = make(map[string]*template.Template)
	m.mu.Unlock()
}

// ClearInlineCache clears only the inline template cache.
func (m *TemplateManager) ClearInlineCache() {
	m.mu.Lock()
	m.inlineCache = make(map[string]*template.Template)
	m.mu.Unlock()
}
