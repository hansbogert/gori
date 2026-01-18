# Gori TODO & Roadmap

This document contains a comprehensive list of improvements, bug fixes, and feature enhancements for the Gori project. It's designed to be used by both human contributors and AI agents working on this repository.

## 🚨 CRITICAL ISSUES (Must Fix First)



### 3. **Missing Error Handling**
**Status**: 🟡 HIGH  
**File**: `cmd/gori.go` - `getLikelyUpstreamMainishBranch()`  
**Issue**: Silent failures and inconsistent error propagation  
**Fix**: Implement proper error wrapping with context  
**Impact**: Poor user experience and debugging difficulties


\n## 🟡 LOW PRIORITY ISSUES\n\n### 1. **Installation Method**\n**Status**: 🟡 LOW\n**Issue**: Current manual clone/install method works fine but could be streamlined\n**Current Method**:\n```bash\ngit clone github.com:hansbogert/gori.git && cd gori\ngo install ./cmd/gori.go\n```\n**Rationale**: Current 2-line method is adequate for target Go developer audience\n**Future Consideration**: If user base expands beyond Go developers, could revisit\n
## 🏗️ ARCHITECTURAL IMPROVEMENTS

### 4. **Code Organization**
**Status**: 🟡 HIGH  
**Current Problem**: 379-line monolithic `cmd/gori.go` with mixed responsibilities  
**Proposed Structure**:
```
├── pkg/
│   ├── git/          # Git operations abstraction
│   ├── config/       # Configuration management
│   ├── scanner/      # Repository scanning logic
│   └── ui/           # User interface helpers
├── cmd/
│   └── gori/         # CLI entry point only
```  
**Files to Refactor**: `cmd/gori.go`, `project.go`, `snooze.go`

### 5. **Interface Design**
**Status**: 🟡 MEDIUM  
**Files**: New files needed  
**Create**: Proper interfaces for git operations to enable testing and flexibility  
**Benefits**: Better testability, easier switching between implementations

### 6. **Configuration Management**
**Status**: 🟡 MEDIUM  
**File**: `snooze.go`  
**Issue**: Basic snooze functionality lacks validation and defaults  
**Fix**: Add configuration validation and default handling  
**Impact**: More robust configuration system

## 🔒 SECURITY & SAFETY

### 7. **Command Injection Risk**
**Status**: 🔴 HIGH  
**File**: `cmd/gori.go` - `visitProjects()` function  
**Issue**: Shell execution without proper sanitization  
**Fix**: Implement proper shell escaping and input validation  
**Priority**: Critical security vulnerability

### 8. **Path Validation**
**Status**: 🟡 MEDIUM  
**Issue**: No validation of repository paths  
**Fix**: Add path sandboxing and validation  
**Impact**: Prevents potential path traversal attacks

### 9. **Configuration File Security**
**Status**: 🟡 MEDIUM  
**File**: Configuration parsing in CUE files  
**Issue**: CUE parsing errors not properly handled  
**Fix**: Implement proper error handling for config files

## ⚡ PERFORMANCE OPTIMIZATIONS

### 10. **Git Operations Performance**
**Status**: 🟡 HIGH  
**Current Problem**: go-git status operations are very slow  
**Documented in**: README.md lines 47-55  
**Solutions**:
- Implement status caching
- Use worker pools more efficiently
- Consider hybrid approach with native git commands  
**Alternative**: Upstream performance fix to go-git

### 11. **Concurrent Processing**
**Status**: 🟡 MEDIUM  
**Issue**: Sequential result processing despite concurrent git operations  
**Fix**: Streamline result collection and processing  
**Files**: Concurrency logic in `cmd/gori.go`

### 12. **File Operations**
**Status**: 🟡 LOW  
**Issue**: Multiple file reads for configuration  
**Fix**: Implement configuration caching  
**Impact**: Minor performance improvement

## 🧪 TESTING & QUALITY

### 13. **Test Coverage Expansion**
**Status**: 🟡 HIGH  
**Current Coverage**: Limited to basic git operations  
**Missing Tests**:
- Snooze functionality (`snooze.go`)
- CLI parsing and flags
- Error handling paths  
- Configuration validation  
- Security features
**Files**: `cmd/main_test.go`, `cmd/script_test.go`

### 14. **Benchmark Tests**
**Status**: 🟡 MEDIUM  
**Issue**: Performance claims are unverified  
**Fix**: Add benchmarks for repository scanning operations  
**Files**: New benchmark files needed

### 15. **Integration Tests**
**Status**: 🟡 MEDIUM  
**Current**: Good test fixtures exist in `test/` directory  
**Enhancement**: Add more edge case scenarios  
**Focus**: Large repositories, network issues, configuration errors

## 📚 DOCUMENTATION & UX

### 16. **README Enhancement**
**Status**: 🟡 MEDIUM  
**File**: `README.md`  
**Missing**:
- Troubleshooting guide
- Contributing guidelines
- Configuration file format documentation  
- API documentation for internal functions

### 17. **Error Message Improvement**
**Status**: 🟡 HIGH  
**Issue**: Cryptic error output for common issues  
**Fix**: Implement user-friendly error messages with actionable suggestions  
**Impact**: Better user experience

### 18. **User Experience Features**
**Status**: 🟡 LOW  
**Missing**:
- Progress indication for long operations
- Verbose/quiet modes
- Customizable emoji indicators
- Better interactive mode feedback

## 🚀 FEATURE ENHANCEMENTS

### 19. **Multi-level Directory Support**
**Status**: 🟡 MEDIUM  
**Current Limitation**: Assumes flat projects dir (mentioned in README.md)  
**Feature**: Support recursive directory scanning  
**Files**: Scanner logic in `cmd/gori.go`

### 20. **Global Configuration**
**Status**: 🟡 MEDIUM  
**Current**: Only per-project configuration via `.goriignore.cue`  
**Feature**: Add global configuration support  
**Integration**: Mentioned in existing todo item

### 21. **Additional Git Operations**
**Status**: 🟡 LOW  
**Features to Add**:
- Pull/push/fetch operations
- Branch switching
- Commit management  
**Integration**: Interactive mode enhancements

### 22. **Export Formats**
**Status**: 🟡 LOW  
**Formats**: JSON, CSV for integration with other tools  
**Use Case**: DevOps pipelines, reporting

### 23. **FZF Integration for Project Selection**
**Status**: 🟡 MEDIUM  
**Current**: Sequential project navigation in interactive mode  
**Feature**: Replace sequential navigation with fzf-based fuzzy selection  
**Benefits**: Faster project selection, better UX for large project lists  
**Files**: `cmd/gori.go` - `visitProjects()` function  
**Implementation**: Integrate fzf for interactive project list selection

## 📋 IMPLEMENTATION PHASES

### Phase 1 - Critical Fixes (1-2 days)
**Priority**: 🔴 CRITICAL  
**Tasks**:
- [ ] Add proper error handling in `getLikelyUpstreamMainishBranch()`
- [ ] Implement input validation for shell commands
- [ ] Add basic security measures

**Files**: `cmd/gori.go`, `go.mod`

### Phase 2 - Architecture (3-5 days)
**Priority**: 🟡 HIGH  
**Tasks**:
- [ ] Extract business logic into separate packages
- [ ] Implement proper interfaces for git operations
- [ ] Add comprehensive test coverage
- [ ] Improve documentation
- [ ] Refactor configuration system

**Files**: New package structure, existing files refactoring

### Phase 3 - Features (1-2 weeks)
**Priority**: 🟡 MEDIUM  
**Tasks**:
- [ ] Performance optimizations for git operations
- [ ] Enhanced configuration management
- [ ] Better UX (progress bars, error messages)
- [ ] Multi-level directory support
- [ ] Additional git operations
- [ ] FZF integration for project selection

**Files**: Multiple files across the project

### Phase 4 - Advanced Features (2-3 weeks)
**Priority**: 🟢 LOW  
**Tasks**:
- [ ] Plugin system for custom status indicators
- [ ] Export formats (JSON, CSV)
- [ ] Web interface (mentioned in existing TODO)
- [ ] Server/client architecture (existing TODO item)

## FOR AI AGENTS

### How to Use This TODO
1. **Always read this TODO first** before making any changes to the codebase
2. **Follow implementation phases in order** - Phase 1 must be completed before Phase 2
3. **Update items as you complete them** - change status, add notes, mark as done
4. **Consider dependencies** - some items depend on others being completed first
5. **Test your changes** - run the provided testing commands before considering a task complete

### Current Project State
- **Test Failures**: Tests fail due to nil pointer dereference in isBranchUpstreamed function`
- **Installation**: Broken due to custom fork dependency in `go.mod`
- **Tests**: Cannot run due to compilation failure
- **Documentation**: Basic but functional README with clear installation and usage instructions
- **Dependencies**: Uses custom fork of go-git for performance reasons

### Testing Commands
```bash
# Build the project
go build ./cmd/gori.go

# Run all tests (currently failing due to test panics)
go test ./...

# Run integration tests
go test ./cmd/ -v

# Install locally (will fail until go.mod is fixed)
go install ./cmd/gori.go
```

### Coding Standards
- **Go Formatting**: Always run `go fmt` before committing
- **Error Handling**: Use proper error wrapping with context (`fmt.Errorf` or `errors.Wrap`)
- **Naming**: Follow Go conventions - camelCase for variables, PascalCase for exported types
- **Comments**: Add comments for all public functions and complex logic
- **Testing**: Write tests for all new functionality, aim for good coverage
- **Dependencies**: Prefer standard library and minimal external dependencies

### Important Files to Understand
- `cmd/gori.go` - Main CLI application (379 lines, monolithic)
- `project.go` - ProjectStatus struct and basic methods
- `snooze.go` - Ignore/snooze functionality (255 lines)
- `go.mod` - Dependencies and module definition
- `README.md` - User documentation
- `todo.md` - Original TODO items (integrated into this file)

### Development Workflow
1. **Fix Phase 1 issues first** - nothing else matters until the tool compiles and runs
2. **Create branches** for each major feature or phase
3. **Write tests** before implementing new functionality
4. **Update this TODO** as you complete items
5. **Commit frequently** with descriptive messages following Go conventions

---

## Legacy TODO Items (Integrated Above)

The following items from the original `todo.md` have been integrated into the structured sections above:

- [x] ~~use git config's pushDefault, instead of assuming origin~~ → **Moved to Feature Enhancements (Phase 3)**
- [x] ~~allow the use of predefined directories in a global cue config~~ → **Moved to Feature Enhancements (Phase 3)**  
- [x] ~~add support for server/client setup where the server continously updates by using inotify and comparable mechanisms on osx and windows~~ → **Moved to Advanced Features (Phase 4)**

---

*This TODO is a living document. Please update it as you work on the project and add new items as needed.*
