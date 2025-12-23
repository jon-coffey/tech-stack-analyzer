package parsers

import (
	"encoding/json"
	"strings"

	"github.com/petrarca/tech-stack-analyzer/internal/types"
)

// PackageLockJSON represents the structure of package-lock.json
// Enhanced with deps.dev patterns for comprehensive dependency analysis
type PackageLockJSON struct {
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	LockfileVersion int                    `json:"lockfileVersion"`
	Packages        map[string]PackageInfo `json:"packages"`
	Dependencies    map[string]PackageInfo `json:"dependencies,omitempty"` // v2 format
}

// PackageInfo represents a package in package-lock.json
// Enhanced with deps.dev patterns for better dependency classification
type PackageInfo struct {
	Version      string                 `json:"version"`
	Resolved     string                 `json:"resolved,omitempty"`
	Link         bool                   `json:"link,omitempty"`
	Dev          bool                   `json:"dev,omitempty"`
	Optional     bool                   `json:"optional,omitempty"`
	Bundled      bool                   `json:"bundled,omitempty"`
	Dependencies map[string]PackageInfo `json:"dependencies,omitempty"`
}

// ParsePackageLockOptions contains configuration options for ParsePackageLock
type ParsePackageLockOptions struct {
	IncludeTransitive bool // Include transitive dependencies (default: false for backward compatibility)
}

// ParsePackageLock parses package-lock.json content and returns comprehensive dependencies
// Enhanced with deps.dev patterns for transitive dependency analysis and scope detection
func ParsePackageLock(content []byte, packageJSON *PackageJSON) []types.Dependency {
	return ParsePackageLockWithOptions(content, packageJSON, ParsePackageLockOptions{})
}

// ParsePackageLockWithOptions parses package-lock.json content with configurable options
// Enhanced with deps.dev patterns for transitive dependency analysis and scope detection
func ParsePackageLockWithOptions(content []byte, packageJSON *PackageJSON, options ParsePackageLockOptions) []types.Dependency {
	var lockfile PackageLockJSON
	if err := json.Unmarshal(content, &lockfile); err != nil {
		return nil
	}

	// Build maps of direct dependency names with their scopes from package.json
	prodDeps := make(map[string]bool)
	devDeps := make(map[string]bool)
	peerDeps := make(map[string]bool)
	optionalDeps := make(map[string]bool)

	// Use the enhanced PackageJSON if available for better scope detection
	if packageJSON != nil {
		for name := range packageJSON.Dependencies {
			prodDeps[name] = true
		}
		for name := range packageJSON.DevDependencies {
			devDeps[name] = true
		}
		// Try to detect peer and optional dependencies if enhanced struct is available
		if enhancedPkg, err := parseEnhancedPackageJSON(content); err == nil {
			for name := range enhancedPkg.PeerDependencies {
				peerDeps[name] = true
			}
			for name := range enhancedPkg.OptionalDependencies {
				optionalDeps[name] = true
			}
		}
	}

	var dependencies []types.Dependency

	// Handle both v2 (dependencies) and v3+ (packages) lockfile formats
	if len(lockfile.Packages) > 0 {
		// v3+ format with packages field
		for path, pkg := range lockfile.Packages {
			if path == "" {
				continue // Skip root package
			}

			name := extractNameFromNodeModulesPath(path)
			if name == "" {
				continue
			}

			// Skip bundled dependencies (deps.dev pattern)
			if pkg.Bundled {
				continue
			}

			// Filter transitive dependencies based on options (default: false for backward compatibility)
			if !options.IncludeTransitive && strings.Count(path, "node_modules/") != 1 {
				continue
			}

			// Determine scope based on package.json and lockfile metadata
			scope := determineScopeFromLockfile(name, pkg, prodDeps, devDeps, peerDeps, optionalDeps)

			dependencies = append(dependencies, types.Dependency{
				Type:       "npm",
				Name:       name,
				Version:    pkg.Version,
				SourceFile: "package-lock.json",
				Scope:      scope,
			})
		}
	} else if len(lockfile.Dependencies) > 0 {
		// v2 format with dependencies field (recursive parsing)
		if options.IncludeTransitive {
			dependencies = parseDependenciesV2(lockfile.Dependencies, "", prodDeps, devDeps, peerDeps, optionalDeps)
		} else {
			// For backward compatibility, only return top-level dependencies
			for name, dep := range lockfile.Dependencies {
				if dep.Bundled {
					continue
				}
				scope := determineScopeFromLockfile(name, PackageInfo{Dev: dep.Dev, Optional: dep.Optional}, prodDeps, devDeps, peerDeps, optionalDeps)
				dependencies = append(dependencies, types.Dependency{
					Type:       "npm",
					Name:       name,
					Version:    dep.Version,
					SourceFile: "package-lock.json",
					Scope:      scope,
				})
			}
		}
	}

	return dependencies
}

// extractNameFromNodeModulesPath extracts package name from package-lock.json path
// e.g., "node_modules/express" -> "express"
// e.g., "node_modules/@babel/core" -> "@babel/core"
// e.g., "node_modules/express/node_modules/accepts" -> "accepts"
func extractNameFromNodeModulesPath(path string) string {
	// Split by "node_modules/" to get all segments
	segments := strings.Split(path, "node_modules/")

	if len(segments) < 2 {
		return ""
	}

	// Get the last segment (the actual package)
	lastSegment := segments[len(segments)-1]
	lastSegment = strings.TrimSpace(lastSegment)

	if lastSegment == "" {
		return ""
	}

	// Handle scoped packages like @babel/core
	if strings.HasPrefix(lastSegment, "@") {
		parts := strings.Split(lastSegment, "/")
		if len(parts) >= 2 {
			return strings.Join(parts[:2], "/")
		}
	}

	// Handle regular packages - just return the first part
	parts := strings.Split(lastSegment, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return lastSegment
}

// parseEnhancedPackageJSON attempts to parse package.json with enhanced fields
func parseEnhancedPackageJSON(content []byte) (*PackageJSONEnhanced, error) {
	var enhancedPkg PackageJSONEnhanced
	if err := json.Unmarshal(content, &enhancedPkg); err != nil {
		return nil, err
	}
	return &enhancedPkg, nil
}

// parseDependenciesV2 recursively parses dependencies in v2 lockfile format
// Based on deps.dev patterns for recursive dependency tree traversal
func parseDependenciesV2(
	deps map[string]PackageInfo,
	path string,
	prodDeps, devDeps, peerDeps, optionalDeps map[string]bool,
) []types.Dependency {
	var dependencies []types.Dependency

	for name, dep := range deps {
		// Skip bundled dependencies (deps.dev pattern)
		if dep.Bundled {
			continue
		}

		// Determine scope
		scope := determineScopeFromLockfile(name, dep, prodDeps, devDeps, peerDeps, optionalDeps)

		dependencies = append(dependencies, types.Dependency{
			Type:       "npm",
			Name:       name,
			Version:    dep.Version,
			SourceFile: "package-lock.json",
			Scope:      scope,
		})

		// Recursively parse nested dependencies
		if len(dep.Dependencies) > 0 {
			nestedPath := path + "node_modules/" + name + "/"
			nestedDeps := parseDependenciesV2(dep.Dependencies, nestedPath,
				prodDeps, devDeps, peerDeps, optionalDeps)
			dependencies = append(dependencies, nestedDeps...)
		}
	}

	return dependencies
}

// determineScopeFromLockfile determines the dependency scope based on package.json and lockfile metadata
// Enhanced with deps.dev patterns for accurate scope classification
func determineScopeFromLockfile(
	name string,
	pkg PackageInfo,
	prodDeps, devDeps, peerDeps, optionalDeps map[string]bool,
) string {
	// Check if it's a peer dependency
	if peerDeps[name] {
		return "peer"
	}

	// Check if it's an optional dependency
	if optionalDeps[name] || pkg.Optional {
		return "optional"
	}

	// Check if it's a development dependency
	if devDeps[name] || pkg.Dev {
		return "dev"
	}

	// Check if it's a production dependency
	if prodDeps[name] {
		return "prod"
	}

	// Default to production if not explicitly classified
	// This is a reasonable default for transitive dependencies
	return "prod"
}

// GetLockfileVersion detects the package-lock.json version format
func GetLockfileVersion(content []byte) int {
	var lockfile struct {
		LockfileVersion int `json:"lockfileVersion"`
	}

	if err := json.Unmarshal(content, &lockfile); err != nil {
		return 1 // Default to v1 if parsing fails
	}

	return lockfile.LockfileVersion
}
