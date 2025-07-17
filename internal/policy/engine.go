package policy

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/takakrypt/transparent-encryption/internal/config"
)

type Engine struct {
	config *config.Config
	
	userSetMap     map[string]*config.UserSet
	processSetMap  map[string]*config.ProcessSet
	resourceSetMap map[string]*config.ResourceSet
	policyMap      map[string]*config.Policy
}

type AccessRequest struct {
	Path      string
	Action    string
	UID       int
	GID       int
	ProcessID int
	Binary    string
}

type AccessResult struct {
	Permission string
	ApplyKey   bool
	Audit      bool
	RuleID     string
}

func NewEngine(cfg *config.Config) *Engine {
	engine := &Engine{
		config:         cfg,
		userSetMap:     make(map[string]*config.UserSet),
		processSetMap:  make(map[string]*config.ProcessSet),
		resourceSetMap: make(map[string]*config.ResourceSet),
		policyMap:      make(map[string]*config.Policy),
	}

	for i := range cfg.UserSets {
		engine.userSetMap[cfg.UserSets[i].Code] = &cfg.UserSets[i]
	}

	for i := range cfg.ProcessSets {
		engine.processSetMap[cfg.ProcessSets[i].Code] = &cfg.ProcessSets[i]
	}

	for i := range cfg.ResourceSets {
		engine.resourceSetMap[cfg.ResourceSets[i].Code] = &cfg.ResourceSets[i]
	}

	for i := range cfg.Policies {
		engine.policyMap[cfg.Policies[i].Code] = &cfg.Policies[i]
	}

	return engine
}

func (e *Engine) EvaluateAccess(req *AccessRequest) (*AccessResult, error) {
	log.Printf("[POLICY] EvaluateAccess: path=%s, action=%s, uid=%d, gid=%d, pid=%d, binary=%s", req.Path, req.Action, req.UID, req.GID, req.ProcessID, req.Binary)

	guardPoint := e.findGuardPoint(req.Path)
	if guardPoint == nil {
		log.Printf("[POLICY] No guard point found for path: %s", req.Path)
		return &AccessResult{
			Permission: "permit",
			ApplyKey:   false,
			Audit:      false,
		}, nil
	}

	log.Printf("[POLICY] Found guard point: %s -> %s (policy: %s, enabled: %v)", guardPoint.ProtectedPath, guardPoint.SecureStoragePath, guardPoint.Policy, guardPoint.Enabled)

	if !guardPoint.Enabled {
		log.Printf("[POLICY] Guard point disabled, permitting access")
		return &AccessResult{
			Permission: "permit",
			ApplyKey:   false,
			Audit:      false,
		}, nil
	}

	policy := e.policyMap[guardPoint.Policy]
	if policy == nil {
		log.Printf("[POLICY] Policy %s not found for guard point %s", guardPoint.Policy, guardPoint.Code)
		return nil, fmt.Errorf("policy %s not found for guard point %s", guardPoint.Policy, guardPoint.Code)
	}

	log.Printf("[POLICY] Found policy: %s with %d rules", policy.Name, len(policy.SecurityRules))

	rules := make([]config.SecurityRule, len(policy.SecurityRules))
	copy(rules, policy.SecurityRules)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Order < rules[j].Order
	})

	for _, rule := range rules {
		log.Printf("[POLICY] Evaluating rule %d: %s (userSet: %v, processSet: %v, resourceSet: %v, action: %v)", rule.Order, rule.ID, rule.UserSet, rule.ProcessSet, rule.ResourceSet, rule.Action)
		if e.matchesRule(req, &rule) {
			log.Printf("[POLICY] Rule matched! Permission: %s, ApplyKey: %v, Audit: %v", rule.Effect.Permission, rule.Effect.Option.ApplyKey, rule.Effect.Option.Audit)
			return &AccessResult{
				Permission: rule.Effect.Permission,
				ApplyKey:   rule.Effect.Option.ApplyKey,
				Audit:      rule.Effect.Option.Audit,
				RuleID:     rule.ID,
			}, nil
		}
		log.Printf("[POLICY] Rule did not match")
	}

	log.Printf("[POLICY] No rules matched, using default deny")
	return &AccessResult{
		Permission: "deny",
		ApplyKey:   false,
		Audit:      true,
		RuleID:     "default-deny",
	}, nil
}

func (e *Engine) findGuardPoint(path string) *config.GuardPoint {
	var bestMatch *config.GuardPoint
	maxDepth := -1

	for i := range e.config.GuardPoints {
		gp := &e.config.GuardPoints[i]
		if e.pathMatches(path, gp.ProtectedPath) {
			depth := strings.Count(gp.ProtectedPath, "/")
			if depth > maxDepth {
				maxDepth = depth
				bestMatch = gp
			}
		}
	}

	return bestMatch
}

func (e *Engine) pathMatches(filePath, guardPath string) bool {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	absGuardPath, err := filepath.Abs(guardPath)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absGuardPath, absFilePath)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..")
}

func (e *Engine) matchesRule(req *AccessRequest, rule *config.SecurityRule) bool {
	// Handle browsing (directory listing) separately
	if req.Action == "browse" {
		log.Printf("[POLICY] Checking browsing permission: req.Action=%s, rule.Browsing=%v", req.Action, rule.Browsing)
		if !rule.Browsing {
			log.Printf("[POLICY] Browsing not allowed")
			return false
		}
		log.Printf("[POLICY] Browsing allowed")
	} else {
		// Handle other actions (read, write, etc.)
		log.Printf("[POLICY] Checking action match: req.Action=%s, rule.Action=%v", req.Action, rule.Action)
		if !e.matchesAction(req.Action, rule.Action) {
			log.Printf("[POLICY] Action does not match")
			return false
		}
		log.Printf("[POLICY] Action matches")
	}

	if len(rule.UserSet) > 0 {
		log.Printf("[POLICY] Checking user set match: req.UID=%d, rule.UserSet=%v", req.UID, rule.UserSet)
		if !e.matchesUserSet(req, rule.UserSet) {
			log.Printf("[POLICY] User set does not match")
			return false
		}
		log.Printf("[POLICY] User set matches")
	}

	if len(rule.ProcessSet) > 0 {
		log.Printf("[POLICY] Checking process set match: req.Binary=%s, rule.ProcessSet=%v", req.Binary, rule.ProcessSet)
		if !e.matchesProcessSet(req, rule.ProcessSet) {
			log.Printf("[POLICY] Process set does not match")
			return false
		}
		log.Printf("[POLICY] Process set matches")
	}

	if len(rule.ResourceSet) > 0 {
		log.Printf("[POLICY] Checking resource set match: req.Path=%s, rule.ResourceSet=%v", req.Path, rule.ResourceSet)
		if !e.matchesResourceSet(req, rule.ResourceSet) {
			log.Printf("[POLICY] Resource set does not match")
			return false
		}
		log.Printf("[POLICY] Resource set matches")
	}

	log.Printf("[POLICY] All conditions match for rule %s", rule.ID)
	return true
}

func (e *Engine) matchesAction(reqAction string, ruleActions []string) bool {
	for _, action := range ruleActions {
		if action == "all_ops" || action == reqAction {
			return true
		}
		// Handle browse/browsing aliases
		if (reqAction == "browse" && action == "browsing") || (reqAction == "browsing" && action == "browse") {
			return true
		}
	}
	return false
}

func (e *Engine) matchesUserSet(req *AccessRequest, userSets []string) bool {
	for _, userSetCode := range userSets {
		userSet := e.userSetMap[userSetCode]
		if userSet != nil {
			for _, user := range userSet.Users {
				if user.UID == req.UID {
					return true
				}
			}
		}
	}
	return false
}

func (e *Engine) matchesProcessSet(req *AccessRequest, processSets []string) bool {
	for _, processSetCode := range processSets {
		processSet := e.processSetMap[processSetCode]
		if processSet != nil {
			for _, resource := range processSet.ResourceSetList {
				if e.matchesProcessResource(req, &resource) {
					return true
				}
			}
		}
	}
	return false
}

func (e *Engine) matchesProcessResource(req *AccessRequest, resource *config.ProcessSetResource) bool {
	binaryPath := filepath.Join(resource.Directory, resource.File)
	return req.Binary == binaryPath || filepath.Base(req.Binary) == resource.File
}

func (e *Engine) matchesResourceSet(req *AccessRequest, resourceSets []string) bool {
	for _, resourceSetCode := range resourceSets {
		resourceSet := e.resourceSetMap[resourceSetCode]
		if resourceSet != nil {
			for _, resource := range resourceSet.ResourceList {
				if e.matchesResource(req.Path, &resource) {
					return true
				}
			}
		}
	}
	return false
}

func (e *Engine) matchesResource(path string, resource *config.Resource) bool {
	// Get guard point for this path
	guardPoint := e.findGuardPoint(path)
	if guardPoint == nil {
		log.Printf("[POLICY] No guard point found for resource matching: %s", path)
		return false
	}

	// Get relative path within the guard point
	relPath, err := filepath.Rel(guardPoint.ProtectedPath, path)
	if err != nil {
		log.Printf("[POLICY] Failed to get relative path: %s relative to %s", path, guardPoint.ProtectedPath)
		return false
	}

	log.Printf("[POLICY] Resource matching: path=%s, guardPoint=%s, relPath=%s, resource.Directory=%s", path, guardPoint.ProtectedPath, relPath, resource.Directory)

	// Check directory match
	resourceDir := strings.TrimPrefix(resource.Directory, "/")
	log.Printf("[POLICY] Checking directory match: relPath=%s, resourceDir=%s, subfolder=%v", relPath, resourceDir, resource.Subfolder)
	
	// Handle guard point root directory listing
	if relPath == "." {
		// For directory listing of guard point root, allow if any resource in this directory
		log.Printf("[POLICY] Guard point root directory listing - allowing access")
		return true
	}
	
	if resource.Directory != "" {
		// Resource is in a subdirectory
		if resource.Subfolder {
			// Check if path is under this directory
			if !strings.HasPrefix(relPath, resourceDir) {
				log.Printf("[POLICY] Path not under resource directory")
				return false
			}
		} else {
			// Check if path is directly in this directory
			pathDir := filepath.Dir(relPath)
			if pathDir != resourceDir {
				log.Printf("[POLICY] Path not in resource directory")
				return false
			}
		}
	} else {
		// Resource is in guard point root (resource.Directory == "")
		log.Printf("[POLICY] Resource is in guard point root")
		if !resource.Subfolder {
			// File must be directly in guard point root
			pathDir := filepath.Dir(relPath)
			if pathDir != "." {
				log.Printf("[POLICY] File not in guard point root")
				return false
			}
		}
		// If subfolder=true, allow files anywhere in guard point
	}

	// Check file pattern match
	if resource.File != "*" {
		filename := filepath.Base(relPath)
		matched, _ := filepath.Match(resource.File, filename)
		return matched
	}

	return true
}

func (e *Engine) GetProcessInfo(pid int) (string, error) {
	exePath := fmt.Sprintf("/proc/%d/exe", pid)
	binary, err := os.Readlink(exePath)
	if err != nil {
		return "", fmt.Errorf("failed to read process binary: %w", err)
	}
	return binary, nil
}

func (e *Engine) GetProcessUID(pid int) (int, error) {
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return -1, fmt.Errorf("failed to read process status: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				uid, err := strconv.Atoi(fields[1])
				if err != nil {
					return -1, err
				}
				return uid, nil
			}
		}
	}

	return -1, fmt.Errorf("uid not found in process status")
}