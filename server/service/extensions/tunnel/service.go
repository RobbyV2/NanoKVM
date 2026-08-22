package tunnel

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	maxBinarySize   int64 = 64 << 20
	maxLogLines           = 200
	elfHeaderSize         = 20
	elfMachineRiscv       = 243
	elfClass64            = 2
)

var (
	secretKeyMarkers = []string{"SECRET", "KEY", "TOKEN", "PASSWORD"}
	elfMagic         = []byte{0x7f, 'E', 'L', 'F'}
)

type Service struct {
	name proto.TunnelName
}

func NewService(name proto.TunnelName) *Service {
	return &Service{name: name}
}

func (s *Service) GetStatus(c *gin.Context) {
	var rsp proto.Response

	state, message := currentState(s.name)
	pid, _ := pidOf(s.name)

	rsp.OkRspWithData(c, &proto.GetTunnelStatusRsp{
		State:   state,
		Message: message,
		Pid:     pid,
		Custom:  isCustom(s.name),
		Enabled: isEnabled(s.name),
	})

	log.Debugf("get %s status successfully", s.name)
}

func (s *Service) GetConfig(c *gin.Context) {
	var rsp proto.Response

	cfg, err := loadConfig(s.name)
	if err != nil {
		rsp.ErrRsp(c, -1, "get config failed")
		log.Errorf("failed to read %s config: %s", s.name, err)
		return
	}

	rsp.OkRspWithData(c, &proto.GetTunnelConfigRsp{
		Args: cfg.Args,
		Env:  maskEnv(s.name, cfg.Env),
	})

	log.Debugf("get %s config successfully", s.name)
}

func (s *Service) SetConfig(c *gin.Context) {
	var rsp proto.Response

	var req proto.SetTunnelConfigReq
	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if _, err := tokenize(req.Args); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		log.Errorf("failed to parse %s arguments: %s", s.name, err)
		return
	}

	for key := range req.Env {
		if !isValidEnvKey(key) {
			rsp.ErrRsp(c, -1, "invalid environment variable name")
			log.Errorf("rejected %s environment variable name %q", s.name, key)
			return
		}
	}

	current, err := loadConfig(s.name)
	if err != nil {
		rsp.ErrRsp(c, -1, "set config failed")
		log.Errorf("failed to read %s config: %s", s.name, err)
		return
	}

	cfg := Config{
		Args: req.Args,
		Env:  mergeEnv(current.Env, req.Env),
	}

	if err := saveConfig(s.name, cfg); err != nil {
		rsp.ErrRsp(c, -1, "set config failed")
		log.Errorf("failed to save %s config: %s", s.name, err)
		return
	}

	if isInstalled(s.name) {
		if err := writeWrapper(s.name, cfg); err != nil {
			rsp.ErrRsp(c, -2, "set config failed")
			log.Errorf("failed to write %s wrapper: %s", s.name, err)
			return
		}
	}

	rsp.OkRsp(c)
	log.Debugf("set %s config successfully", s.name)
}

func (s *Service) Start(c *gin.Context) {
	s.lifecycle(c, "start")
}

func (s *Service) Restart(c *gin.Context) {
	s.lifecycle(c, "restart")
}

func (s *Service) lifecycle(c *gin.Context, action string) {
	var rsp proto.Response

	cfg, err := loadConfig(s.name)
	if err != nil {
		rsp.ErrRsp(c, -1, action+" failed")
		log.Errorf("failed to read %s config: %s", s.name, err)
		return
	}

	if err := writeWrapper(s.name, cfg); err != nil {
		rsp.ErrRsp(c, -1, action+" failed")
		log.Errorf("failed to prepare %s wrapper: %s", s.name, err)
		return
	}

	if err := enableInitScript(s.name); err != nil {
		rsp.ErrRsp(c, -1, action+" failed")
		log.Errorf("failed to install %s init script: %s", s.name, err)
		return
	}

	if err := runInitScript(s.name, action); err != nil {
		rsp.ErrRsp(c, -2, action+" failed")
		log.Errorf("failed to run %s %s: %s", s.name, action, err)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("%s %s successfully", s.name, action)
}

func (s *Service) Stop(c *gin.Context) {
	var rsp proto.Response

	stopErr := runInitScript(s.name, "stop")

	if err := os.Remove(initScriptPath(s.name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		rsp.ErrRsp(c, -1, "stop failed")
		log.Errorf("failed to remove %s init script: %s", s.name, err)
		return
	}

	if stopErr != nil {
		rsp.ErrRsp(c, -2, "stop failed")
		log.Errorf("failed to run %s stop: %s", s.name, stopErr)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("%s stop successfully", s.name)
}

func (s *Service) GetLogs(c *gin.Context) {
	var rsp proto.Response

	rsp.OkRspWithData(c, &proto.GetTunnelLogsRsp{
		Lines: logTail(s.name, maxLogLines),
	})

	log.Debugf("get %s logs successfully", s.name)
}

func (s *Service) UploadBinary(c *gin.Context) {
	var rsp proto.Response

	if _, running := pidOf(s.name); running {
		rsp.ErrRsp(c, -1, "stop the service before uploading")
		return
	}

	if err := uploadBinary(c, s.name); err != nil {
		rsp.ErrRsp(c, -2, fmt.Sprintf("upload failed: %s", err))
		log.Errorf("failed to upload %s binary: %s", s.name, err)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("upload %s binary successfully", s.name)
}

func (s *Service) DeleteBinary(c *gin.Context) {
	var rsp proto.Response

	if _, running := pidOf(s.name); running {
		rsp.ErrRsp(c, -1, "stop the service before deleting")
		return
	}

	if !isCustom(s.name) {
		rsp.OkRsp(c)
		return
	}

	if err := os.Remove(binaryFile(s.name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		rsp.ErrRsp(c, -1, "delete failed")
		log.Errorf("failed to remove %s binary: %s", s.name, err)
		return
	}

	if err := setCustom(s.name, false); err != nil {
		rsp.ErrRsp(c, -1, "delete failed")
		log.Errorf("failed to clear %s custom marker: %s", s.name, err)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("delete %s binary successfully", s.name)
}

func isSecretKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, marker := range secretKeyMarkers {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func maskEnv(name proto.TunnelName, env map[string]string) []proto.TunnelEnvEntry {
	spec, _ := specOf(name)

	seen := make(map[string]bool, len(env))
	keys := make([]string, 0, len(env)+len(spec.SeededEnv))
	for _, key := range spec.SeededEnv {
		if seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}

	extra := make([]string, 0, len(env))
	for key := range env {
		if seen[key] {
			continue
		}
		seen[key] = true
		extra = append(extra, key)
	}
	sort.Strings(extra)
	keys = append(keys, extra...)

	entries := make([]proto.TunnelEnvEntry, 0, len(keys))
	for _, key := range keys {
		value := env[key]
		secret := isSecretKey(key)

		entry := proto.TunnelEnvEntry{
			Key:        key,
			Value:      value,
			Secret:     secret,
			Configured: value != "",
		}
		if secret {
			entry.Value = ""
		}
		entries = append(entries, entry)
	}

	return entries
}

func mergeEnv(current map[string]string, next map[string]string) map[string]string {
	merged := make(map[string]string, len(next))
	for key, value := range next {
		if value == "" && isSecretKey(key) {
			if kept, ok := current[key]; ok {
				merged[key] = kept
			}
			continue
		}
		merged[key] = value
	}
	return merged
}

func resolveInitScript(name proto.TunnelName) (string, bool) {
	for _, path := range []string{initScriptPath(name), initSeedPath(name)} {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

func enableInitScript(name proto.TunnelName) error {
	source, err := os.Open(initSeedPath(name))
	if err != nil {
		return fmt.Errorf("open tunnel init script: %w", err)
	}
	defer func() { _ = source.Close() }()

	file, err := newAtomicFile(initScriptPath(name), 0o755)
	if err != nil {
		return err
	}
	defer file.Discard()

	if _, err := io.Copy(file, source); err != nil {
		return fmt.Errorf("copy tunnel init script: %w", err)
	}
	return file.Commit()
}

func runInitScript(name proto.TunnelName, action string) error {
	script, ok := resolveInitScript(name)
	if !ok {
		return nil
	}

	output, err := exec.Command(script, action).CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return err
		}
		return fmt.Errorf("%s: %w", trimmed, err)
	}
	return nil
}

func uploadBinary(c *gin.Context, name proto.TunnelName) error {
	expected, err := utils.ParseSHA256Checksum(c.GetHeader("X-SHA256-Checksum"))
	if err != nil {
		return err
	}

	maxRequestBytes := maxBinarySize + (1 << 20)
	if c.Request.ContentLength > maxRequestBytes {
		return fmt.Errorf("upload exceeds %d bytes", maxRequestBytes)
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBytes)

	reader, err := c.Request.MultipartReader()
	if err != nil {
		return fmt.Errorf("invalid multipart data: %w", err)
	}

	saved := false
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read multipart: %w", err)
		}

		if part.FormName() != "file" {
			continue
		}
		if saved {
			return errors.New("multiple files uploaded")
		}

		if err := saveBinary(part, name, expected); err != nil {
			return err
		}
		saved = true
	}

	if !saved {
		return errors.New("no file uploaded")
	}
	return nil
}

func saveBinary(part *multipart.Part, name proto.TunnelName, expected []byte) error {
	if err := utils.ValidateSafeFilename(part.FileName()); err != nil {
		return err
	}

	file, err := newAtomicFile(binaryFile(name), 0o755)
	if err != nil {
		return err
	}
	defer file.Discard()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(part, maxBinarySize+1))
	if err != nil {
		return fmt.Errorf("write tunnel binary: %w", err)
	}
	if written > maxBinarySize {
		return fmt.Errorf("uploaded binary exceeds %d bytes", maxBinarySize)
	}
	if err := file.Flush(); err != nil {
		return err
	}

	if len(expected) > 0 && subtle.ConstantTimeCompare(hasher.Sum(nil), expected) != 1 {
		return errors.New("sha256 checksum mismatch")
	}
	if err := verifyRiscvElf(file.Path()); err != nil {
		return err
	}
	if err := file.Commit(); err != nil {
		return err
	}
	return setCustom(name, true)
}

func verifyRiscvElf(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open uploaded binary: %w", err)
	}
	defer func() { _ = file.Close() }()

	header := make([]byte, elfHeaderSize)
	if _, err := io.ReadFull(file, header); err != nil {
		return errors.New("uploaded file is not a riscv64 elf binary")
	}

	switch {
	case !bytes.Equal(header[:4], elfMagic):
		return errors.New("uploaded file is not an elf binary")
	case header[4] != elfClass64:
		return errors.New("uploaded file is not a 64 bit elf binary")
	case header[18] != elfMachineRiscv:
		return errors.New("uploaded file is not a riscv64 elf binary")
	default:
		return nil
	}
}
