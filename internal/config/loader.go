package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Load(configDir string) (*Config, error) {
	config := &Config{}

	userSets, err := loadUserSets(filepath.Join(configDir, "user_set.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load user sets: %w", err)
	}
	config.UserSets = userSets

	processSets, err := loadProcessSets(filepath.Join(configDir, "process_set.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load process sets: %w", err)
	}
	config.ProcessSets = processSets

	resourceSets, err := loadResourceSets(filepath.Join(configDir, "resource_set.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load resource sets: %w", err)
	}
	config.ResourceSets = resourceSets

	guardPoints, err := loadGuardPoints(filepath.Join(configDir, "guard-point.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load guard points: %w", err)
	}
	config.GuardPoints = guardPoints

	policies, err := loadPolicies(filepath.Join(configDir, "policy.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}
	config.Policies = policies

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func loadUserSets(filename string) ([]UserSet, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var userSets []UserSet
	if err := json.Unmarshal(data, &userSets); err != nil {
		return nil, err
	}

	return userSets, nil
}

func loadProcessSets(filename string) ([]ProcessSet, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var processSets []ProcessSet
	if err := json.Unmarshal(data, &processSets); err != nil {
		return nil, err
	}

	return processSets, nil
}

func loadResourceSets(filename string) ([]ResourceSet, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var resourceSets []ResourceSet
	if err := json.Unmarshal(data, &resourceSets); err != nil {
		return nil, err
	}

	return resourceSets, nil
}

func loadGuardPoints(filename string) ([]GuardPoint, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var guardPoints []GuardPoint
	if err := json.Unmarshal(data, &guardPoints); err != nil {
		return nil, err
	}

	return guardPoints, nil
}

func loadPolicies(filename string) ([]Policy, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var policies []Policy
	if err := json.Unmarshal(data, &policies); err != nil {
		return nil, err
	}

	return policies, nil
}

func validateConfig(config *Config) error {
	policyMap := make(map[string]bool)
	for _, policy := range config.Policies {
		policyMap[policy.Code] = true
	}

	for _, gp := range config.GuardPoints {
		if !policyMap[gp.Policy] {
			return fmt.Errorf("guard point %s references non-existent policy %s", gp.Code, gp.Policy)
		}
	}

	return nil
}