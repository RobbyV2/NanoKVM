package presentation

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const jsonBodyLimit = 1 << 20

type Service struct {
	manager *Manager
	store   *Store
}

type ProfileSummary struct {
	Name           string   `json:"name"`
	BuiltIn        bool     `json:"built_in"`
	Active         bool     `json:"active"`
	Manufacturer   string   `json:"manufacturer"`
	Product        string   `json:"product"`
	Functions      []string `json:"functions"`
	HasDescriptors bool     `json:"has_descriptors"`
}

type Preview struct {
	Valid      bool           `json:"valid"`
	Errors     []string       `json:"errors"`
	Warnings   []string       `json:"warnings"`
	Profile    string         `json:"profile"`
	Functions  []string       `json:"functions"`
	Endpoints  EndpointUse    `json:"endpoints"`
	Headroom   EndpointUse    `json:"headroom"`
	FIFOs      FIFOAssignment `json:"fifos,omitempty"`
	Operations int            `json:"operations"`
	Device     DeviceIdentity `json:"device"`

	// What the apply would do to the linkage that is up now, and where a failed
	// one would land. Both are absent when there is no gadget to compare
	// against; the compile above still stands on its own.
	Apply    *Outcome `json:"apply,omitempty"`
	Rollback *Outcome `json:"rollback,omitempty"`
}

type Status struct {
	Snapshot       Snapshot        `json:"snapshot"`
	Profile        *ProfileSummary `json:"profile,omitempty"`
	LastKnownGood  string          `json:"last_known_good"`
	RollbackTarget string          `json:"rollback_target"`
}

type nameRequest struct {
	Name string `json:"name"`
}

func NewService() *Service {
	manager := GetManager()
	return &Service{manager: manager, store: manager.store}
}

func newService(manager *Manager) *Service {
	return &Service{manager: manager, store: manager.store}
}

func (s *Service) GetStatus(c *gin.Context) {
	snapshot, err := s.manager.Snapshot()
	if err != nil {
		s.fail(c, -1, "read presentation status", err)
		return
	}

	result := Status{Snapshot: snapshot}
	result.LastKnownGood, err = s.store.LastKnownGood()
	if err != nil {
		s.fail(c, -1, "read last-known-good profile", err)
		return
	}
	result.RollbackTarget, err = s.store.Previous()
	if err != nil {
		s.fail(c, -1, "read rollback target", err)
		return
	}
	if snapshot.Active != "" {
		profile, err := s.store.LoadProfile(snapshot.Active)
		if err != nil {
			s.fail(c, -1, "read active profile", err)
			return
		}
		summary := summarizeProfile(profile, snapshot.Active)
		result.Profile = &summary
	}

	var rsp proto.Response
	rsp.OkRspWithData(c, &result)
}

func (s *Service) GetProfiles(c *gin.Context) {
	profiles, err := s.store.Profiles()
	if err != nil {
		s.fail(c, -1, "read presentation profiles", err)
		return
	}
	active, err := s.store.Active()
	if err != nil {
		s.fail(c, -1, "read active profile", err)
		return
	}

	result := make([]ProfileSummary, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, summarizeProfile(profile, active))
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &gin.H{"profiles": result})
}

func (s *Service) GetProfile(c *gin.Context) {
	profile, ok := s.load(c, c.Param("id"))
	if !ok {
		return
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &profile)
}

func (s *Service) CreateProfile(c *gin.Context) {
	var profile Profile
	if err := decodeJSONBody(c, &profile); err != nil {
		s.error(c, -1, err.Error())
		return
	}
	profile.BuiltIn = false
	if err := s.create(profile); err != nil {
		s.error(c, -2, err.Error())
		return
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &profile)
}

func (s *Service) UpdateProfile(c *gin.Context) {
	id := c.Param("id")
	if _, builtIn := builtinByName(id); builtIn {
		s.error(c, -2, "built-in profiles are read-only; clone one to edit it")
		return
	}
	var profile Profile
	if err := decodeJSONBody(c, &profile); err != nil {
		s.error(c, -1, err.Error())
		return
	}
	if profile.Name != id {
		s.error(c, -2, "profile name must match the route")
		return
	}
	if profile.BuiltIn {
		s.error(c, -2, "custom profiles cannot become built-ins")
		return
	}
	existing, ok := s.load(c, id)
	if !ok {
		return
	}
	if existing.BuiltIn {
		s.error(c, -2, "built-in profiles are read-only; clone one to edit it")
		return
	}
	profile.Normalize()
	if err := profile.Validate(); err != nil {
		s.error(c, -2, err.Error())
		return
	}
	if err := s.store.SaveProfile(profile); err != nil {
		s.fail(c, -3, "save profile", err)
		return
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &profile)
}

func (s *Service) DeleteProfile(c *gin.Context) {
	id := c.Param("id")
	if _, builtIn := builtinByName(id); builtIn {
		s.error(c, -2, "built-in profiles cannot be deleted")
		return
	}
	for _, read := range []func() (string, error){s.store.Active, s.store.LastKnownGood} {
		name, err := read()
		if err != nil {
			s.fail(c, -1, "read presentation state", err)
			return
		}
		if name == id {
			s.error(c, -2, "the active or last-known-good profile cannot be deleted")
			return
		}
	}
	if err := s.store.DeleteProfile(id); err != nil {
		s.fail(c, -3, "delete profile", err)
		return
	}
	var rsp proto.Response
	rsp.OkRsp(c)
}

func (s *Service) CloneProfile(c *gin.Context) {
	profile, ok := s.load(c, c.Param("id"))
	if !ok {
		return
	}
	var req nameRequest
	if err := decodeJSONBody(c, &req); err != nil {
		s.error(c, -1, err.Error())
		return
	}
	profile.Name = req.Name
	profile.BuiltIn = false
	if err := s.create(profile); err != nil {
		s.error(c, -2, err.Error())
		return
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &profile)
}

func (s *Service) ValidateProfile(c *gin.Context) {
	profile, ok := s.load(c, c.Param("id"))
	if !ok {
		return
	}
	s.preview(c, profile)
}

func (s *Service) PreviewProfile(c *gin.Context) {
	var profile Profile
	if err := decodeJSONBody(c, &profile); err != nil {
		s.error(c, -1, err.Error())
		return
	}
	s.preview(c, profile)
}

func (s *Service) ApplyProfile(c *gin.Context) {
	var req nameRequest
	if err := decodeJSONBody(c, &req); err != nil {
		s.error(c, -1, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 30*time.Second)
	defer cancel()
	if err := s.manager.Apply(ctx, req.Name); err != nil {
		s.fail(c, -2, "apply profile", err)
		return
	}
	var rsp proto.Response
	rsp.OkRsp(c)
}

// The last-known-good marker advances onto every profile that binds and
// verifies, so the profile it displaced is the one an operator who dislikes the
// gadget they just applied wants back. The apply path runs the transaction, so
// this refuses a loaned UDC and recovers from a rollback that itself fails.
func (s *Service) RollbackProfile(c *gin.Context) {
	name, err := s.store.Previous()
	if err != nil {
		s.fail(c, -1, "read rollback target", err)
		return
	}
	if name == "" {
		s.error(c, -2, "no earlier profile to roll back to")
		return
	}
	active, err := s.store.Active()
	if err != nil {
		s.fail(c, -1, "read active profile", err)
		return
	}
	// A second click lands here: the gadget is already the one the first asked
	// for, so it is answered without unbinding anything.
	if active != name {
		profile, ok := s.load(c, name)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 30*time.Second)
		defer cancel()
		if err := s.manager.ApplyProfile(ctx, profile); err != nil {
			s.fail(c, -3, "roll back to "+name, err)
			return
		}
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &gin.H{"profile": name})
}

func (s *Service) ImportProfile(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, packageArchiveLimit+1<<20)
	file, err := c.FormFile("package")
	if err != nil {
		s.error(c, -1, "package file is required")
		return
	}
	opened, err := file.Open()
	if err != nil {
		s.fail(c, -1, "open package", err)
		return
	}
	data, readErr := io.ReadAll(io.LimitReader(opened, packageArchiveLimit+1))
	closeErr := opened.Close()
	if readErr != nil || closeErr != nil {
		s.fail(c, -1, "read package", errors.Join(readErr, closeErr))
		return
	}
	profile, err := ImportPackage(data)
	if err != nil {
		s.error(c, -2, err.Error())
		return
	}
	if err := s.create(profile); err != nil {
		s.error(c, -2, err.Error())
		return
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &profile)
}

func (s *Service) ExportProfile(c *gin.Context) {
	profile, ok := s.load(c, c.Param("id"))
	if !ok {
		return
	}
	var data bytes.Buffer
	if err := ExportPackage(&data, profile); err != nil {
		s.fail(c, -1, "export profile", err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.nanokvm-profile.zip"`, profile.Name))
	c.Data(http.StatusOK, PackageMediaType, data.Bytes())
}

func (s *Service) create(profile Profile) error {
	profile.Normalize()
	if profile.BuiltIn {
		return errors.New("custom profiles cannot become built-ins")
	}
	if err := profile.Validate(); err != nil {
		return err
	}
	existing, err := s.store.LoadProfile(profile.Name)
	if err != nil {
		return err
	}
	if existing.Name != "" {
		return fmt.Errorf("profile %q already exists", profile.Name)
	}
	return s.store.SaveProfile(profile)
}

func (s *Service) preview(c *gin.Context, profile Profile) {
	result := preview(profile, s.manager)
	var rsp proto.Response
	rsp.OkRspWithData(c, &result)
}

func preview(profile Profile, manager *Manager) Preview {
	profile.Normalize()
	result := Preview{
		Valid:     true,
		Errors:    []string{},
		Warnings:  []string{},
		Profile:   profile.Name,
		Functions: functionNames(profile.Functions),
	}
	plan, err := Compile(profile, manager.caps)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	result.Operations = len(plan.Ops)
	result.Endpoints = plan.Endpoints
	result.Headroom = plan.Endpoints.Headroom(manager.caps)
	result.FIFOs = plan.FIFOs
	result.Device = plan.Device
	result.Warnings = descriptorWarnings(profile)
	result.Apply, result.Rollback = manager.outcomes(plan)
	return result
}

func descriptorWarnings(profile Profile) []string {
	set := profile.Descriptors
	if set == nil {
		return []string{}
	}
	warnings := []string{"Imported descriptor assets are preserved in the package; the composite ConfigFS backend applies the profile fields and function report descriptors."}
	if len(set.Device) == 18 {
		vendor := fmt.Sprintf("0x%04X", binary.LittleEndian.Uint16(set.Device[8:10]))
		product := fmt.Sprintf("0x%04X", binary.LittleEndian.Uint16(set.Device[10:12]))
		if !strings.EqualFold(vendor, profile.Device.VendorID) || !strings.EqualFold(product, profile.Device.ProductID) {
			warnings = append(warnings, fmt.Sprintf("The stored device descriptor identifies %s:%s while the active profile uses %s:%s.", vendor, product, profile.Device.VendorID, profile.Device.ProductID))
		}
	}
	return warnings
}

func summarizeProfile(profile Profile, active string) ProfileSummary {
	return ProfileSummary{
		Name:           profile.Name,
		BuiltIn:        profile.BuiltIn,
		Active:         profile.Name == active,
		Manufacturer:   profile.Device.Manufacturer,
		Product:        profile.Device.Product,
		Functions:      functionNames(profile.Functions),
		HasDescriptors: profile.Descriptors != nil,
	}
}

func functionNames(functions []Function) []string {
	names := make([]string, 0, len(functions))
	for _, function := range functions {
		names = append(names, functionName(function))
	}
	return names
}

func (s *Service) load(c *gin.Context, name string) (Profile, bool) {
	profile, err := s.store.LoadProfile(name)
	if err != nil {
		s.fail(c, -1, "read profile", err)
		return Profile{}, false
	}
	if profile.Name == "" {
		s.error(c, -2, "profile not found")
		return Profile{}, false
	}
	return profile, true
}

func decodeJSONBody(c *gin.Context, dst any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, jsonBodyLimit)
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	if len(data) == 0 {
		return errors.New("request body is empty")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

func (s *Service) fail(c *gin.Context, code int, action string, err error) {
	log.Errorf("presentation: %s: %s", action, err)
	s.error(c, code, action+": "+err.Error())
}

func (s *Service) error(c *gin.Context, code int, message string) {
	var rsp proto.Response
	rsp.ErrRsp(c, code, message)
}
