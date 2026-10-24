package picoclaw

import (
	"errors"
	"strings"

	"NanoKVM-Server/service/picoclaw/modelconfig"

	"github.com/gin-gonic/gin"
)

func isPicoclawModelConfigured(cfg picoclawConfigFile, security picoclawSecurityConfig, modelName string) bool {
	if modelName == "" {
		return false
	}

	// PicoClaw balances requests across every entry sharing the alias, so each
	// of them has to be callable.
	matched := 0
	for _, model := range cfg.ModelList {
		if strings.TrimSpace(model.ModelName) != modelName {
			continue
		}
		hasKey := picoclawEntryHasAPIKeys(security, modelName, matched, model.APIKeys)
		if !modelconfig.Usable(model.Provider, model.Model, model.APIBase, hasKey) {
			return false
		}
		matched++
	}

	return matched > 0
}

// picoclawEntryHasAPIKeys follows PicoClaw's credential lookup for the
// index-th entry named modelName: the "<alias>:<index>" entry of .security.yml,
// then the bare alias, then api_keys in config.json. A present .security.yml
// entry replaces the config.json keys even when it is empty. PicoClaw ignores a
// singular api_key in a version 3 config.json.
func picoclawEntryHasAPIKeys(security picoclawSecurityConfig, modelName string, index int, configKeys []string) bool {
	if entry, ok := security.ModelList[modelconfig.SecurityKey(modelName, index)]; ok {
		return len(entry.APIKeys) > 0
	}
	if entry, ok := security.ModelList[modelName]; ok {
		return len(entry.APIKeys) > 0
	}
	return len(configKeys) > 0
}

type ModelConfigUpdateRequest struct {
	Model   string `json:"model"`
	APIBase string `json:"api_base"`
	APIKey  string `json:"api_key"`
}

func (s *Service) UpdateModelConfig(c *gin.Context) {
	s.ensureDependencies()
	var req ModelConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writePicoclawError(c, newPicoclawError(CodeInvalidAction, "invalid model config payload"))
		return
	}
	// Validate before touching control or the runtime: a form PicoClaw could
	// not use must leave the running assistant alone.
	resolved, err := modelconfig.Resolve(modelconfig.Request{
		Model:   req.Model,
		APIBase: req.APIBase,
		APIKey:  req.APIKey,
	})
	if err != nil {
		var fieldErr *modelconfig.Error
		if errors.As(err, &fieldErr) {
			writePicoclawErrorWithData(c, newPicoclawError(CodeInvalidAction, fieldErr.Message), gin.H{"field": fieldErr.Field})
			return
		}
		writePicoclawError(c, newPicoclawError(CodeInvalidAction, err.Error()))
		return
	}

	currentStatus := s.runtime.Get()
	shouldRestart := currentStatus.Ready || currentStatus.Status == "ready"
	var releaseControl func()
	if shouldRestart {
		// Restart follows StartRuntime's lock order: stable PicoClaw control
		// lease first, then the runtime lifecycle lock around config write and
		// stop/start.
		var controlErr *PicoclawError
		releaseControl, controlErr = s.acquireControlMode()
		if controlErr != nil {
			writePicoclawErrorWithData(c, controlErr, gin.H{"status": s.runtimeStatus()})
			return
		}
		defer releaseControl()
	}

	unlockLifecycle := s.lockRuntimeLifecycle()
	defer unlockLifecycle()
	currentStatus = s.runtime.Get()
	if !shouldRestart && (currentStatus.Ready || currentStatus.Status == "ready") {
		var controlErr *PicoclawError
		releaseControl, controlErr = s.acquireControlMode()
		if controlErr != nil {
			writePicoclawErrorWithData(c, controlErr, gin.H{"status": s.runtimeStatus()})
			return
		}
		defer releaseControl()
		shouldRestart = true
	}

	if err := savePicoclawModelConfig(resolved); err != nil {
		writePicoclawError(c, newPicoclawError(CodeRuntimeUnavailable, err.Error()))
		return
	}
	if err := ensurePicoclawStartupDefaults(); err != nil {
		writePicoclawError(c, newPicoclawError(CodeRuntimeUnavailable, "model config saved, but failed to persist picoclaw defaults: "+err.Error()))
		return
	}

	if shouldRestart {
		if _, _, stopErr := s.stopRuntime(); stopErr != nil {
			writePicoclawError(c, newPicoclawError(CodeRuntimeUnavailable, "model config saved, but failed to restart picoclaw runtime: "+stopErr.Message))
			return
		}
		if _, _, startErr := s.startRuntime(); startErr != nil {
			writePicoclawError(c, newPicoclawError(CodeRuntimeUnavailable, "model config saved, but failed to restart picoclaw runtime: "+startErr.Message))
			return
		}
		s.setRuntimeIntentDesired(true, "model_config")
	} else {
		_ = s.syncConfigFromPicoclaw()
		_ = s.ensureRuntimeReadyForLifecycle()
	}

	writeSuccess(c, gin.H{
		"model_name": resolved.ModelName,
		"model":      resolved.Model,
		"api_base":   resolved.APIBase,
		"status":     s.runtimeStatus(),
	})
}

// savePicoclawModelConfig makes resolved the default model: one model_list
// entry under its alias and its key in .security.yml, initializing PicoClaw's
// configuration first when none exists.
func savePicoclawModelConfig(resolved modelconfig.Resolved) error {
	doc, err := loadOrInitializePicoclawConfigDocument()
	if err != nil {
		return err
	}

	doc.raw["version"] = currentPicoclawConfigVersion

	modelList, _ := doc.raw["model_list"].([]any)
	doc.raw["model_list"] = modelconfig.ApplyEntry(modelList, resolved)

	agents, ok := doc.raw["agents"].(map[string]any)
	if !ok {
		agents = map[string]any{}
		doc.raw["agents"] = agents
	}
	defaults, ok := agents["defaults"].(map[string]any)
	if !ok {
		defaults = map[string]any{}
		agents["defaults"] = defaults
	}
	defaults["model_name"] = resolved.ModelName
	delete(defaults, "model")

	// ApplyEntry leaves exactly one entry under the alias, so its credentials
	// are "<alias>:0". Keys of entries it replaced would otherwise be read for
	// a later entry with the same alias.
	if doc.security.ModelList == nil {
		doc.security.ModelList = map[string]picoclawModelSecurityEntry{}
	}
	for key := range doc.security.ModelList {
		if modelconfig.IsSecurityKeyFor(key, resolved.ModelName) {
			delete(doc.security.ModelList, key)
		}
	}
	entry := picoclawModelSecurityEntry{}
	if resolved.APIKey != "" {
		entry.APIKeys = []string{resolved.APIKey}
	}
	doc.security.ModelList[modelconfig.SecurityKey(resolved.ModelName, 0)] = entry

	// The key is written first, so a failure of either write leaves config.json
	// naming the previous model rather than one without its key.
	if err := doc.saveSecurity(); err != nil {
		return err
	}
	return doc.saveConfig()
}
