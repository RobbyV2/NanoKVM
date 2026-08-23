package presentation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type responseEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func testPresentationService(t *testing.T) (*Service, *RecordOps) {
	t.Helper()
	manager, ops := newTestManager(t)
	return newService(manager), ops
}

func profileRouter(service *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/status", service.GetStatus)
	router.GET("/profiles", service.GetProfiles)
	router.POST("/profiles", service.CreateProfile)
	router.GET("/profiles/:id", service.GetProfile)
	router.PUT("/profiles/:id", service.UpdateProfile)
	router.DELETE("/profiles/:id", service.DeleteProfile)
	router.POST("/profiles/:id/clone", service.CloneProfile)
	router.POST("/profiles/:id/validate", service.ValidateProfile)
	router.PUT("/preview", service.PreviewProfile)
	router.PUT("/apply", service.ApplyProfile)
	router.POST("/rollback", service.RollbackProfile)
	return router
}

func TestProfileCRUDPreservesBuiltIns(t *testing.T) {
	service, _ := testPresentationService(t)
	router := profileRouter(service)

	clone := requestJSON(t, router, http.MethodPost, "/profiles/standard/clone", nameRequest{Name: "desk"})
	if clone.Code != 0 {
		t.Fatalf("clone = %+v", clone)
	}

	profile := decodeData[Profile](t, requestJSON(t, router, http.MethodGet, "/profiles/desk", nil))
	profile.Device.Manufacturer = "RobbyV2"
	profile.Device.Product = "Desk KVM"
	updated := requestJSON(t, router, http.MethodPut, "/profiles/desk", profile)
	if updated.Code != 0 {
		t.Fatalf("update = %+v", updated)
	}

	listed := decodeData[struct {
		Profiles []ProfileSummary `json:"profiles"`
	}](t, requestJSON(t, router, http.MethodGet, "/profiles", nil))
	if len(listed.Profiles) != 3 || listed.Profiles[0].Name != ProfileHIDOnly || listed.Profiles[1].Name != ProfileStandard {
		t.Fatalf("profiles = %+v", listed.Profiles)
	}

	deleted := requestJSON(t, router, http.MethodDelete, "/profiles/desk", nil)
	if deleted.Code != 0 {
		t.Fatalf("delete = %+v", deleted)
	}
	if got := requestJSON(t, router, http.MethodDelete, "/profiles/standard", nil); got.Code == 0 {
		t.Fatal("built-in profile deleted")
	}
	standard := decodeData[Profile](t, requestJSON(t, router, http.MethodGet, "/profiles/standard", nil))
	if standard.Device.Product != "NanoKVM" || !standard.BuiltIn {
		t.Fatalf("standard = %+v", standard)
	}
}

func TestPreviewIsPureAndApplyChangesState(t *testing.T) {
	service, ops := testPresentationService(t)
	router := profileRouter(service)
	profile := standardProfile()
	profile.Name = "desk"
	profile.BuiltIn = false

	before := len(ops.Trace())
	previewRsp := requestJSON(t, router, http.MethodPut, "/preview", profile)
	preview := decodeData[Preview](t, previewRsp)
	if previewRsp.Code != 0 || !preview.Valid || preview.Operations == 0 {
		t.Fatalf("preview = %+v envelope = %+v", preview, previewRsp)
	}
	if after := len(ops.Trace()); after != before {
		t.Fatalf("preview emitted %d ops", after-before)
	}
	if saved, err := service.store.LoadProfile("desk"); err != nil || saved.Name != "" {
		t.Fatalf("preview persisted profile: %+v %v", saved, err)
	}

	if created := requestJSON(t, router, http.MethodPost, "/profiles", profile); created.Code != 0 {
		t.Fatalf("create = %+v", created)
	}
	if applied := requestJSON(t, router, http.MethodPut, "/apply", nameRequest{Name: "desk"}); applied.Code != 0 {
		t.Fatalf("apply = %+v", applied)
	}
	status := decodeData[Status](t, requestJSON(t, router, http.MethodGet, "/status", nil))
	if status.Snapshot.Active != "desk" || status.LastKnownGood != "desk" || status.Profile == nil || status.Profile.Name != "desk" {
		t.Fatalf("status = %+v", status)
	}
}

func TestPreviewReturnsValidationWithoutMutation(t *testing.T) {
	service, ops := testPresentationService(t)
	router := profileRouter(service)
	profile := standardProfile()
	profile.Name = "bad/name"
	profile.BuiltIn = false

	before := len(ops.Trace())
	result := decodeData[Preview](t, requestJSON(t, router, http.MethodPut, "/preview", profile))
	if result.Valid || len(result.Errors) != 1 {
		t.Fatalf("preview = %+v", result)
	}
	if after := len(ops.Trace()); after != before {
		t.Fatalf("invalid preview emitted %d ops", after-before)
	}
}

func TestIncompatibleProfileCanBeStoredButNotApplied(t *testing.T) {
	service, ops := testPresentationService(t)
	router := profileRouter(service)
	profile := standardProfile()
	profile.Name = "endpoint-heavy"
	profile.BuiltIn = false
	profile.Functions = append(profile.Functions, Function{
		Kind:     FunctionNCM,
		Instance: "usb1",
		Net:      &NetFunction{CompatibleID: "WINNCM"},
	}, Function{
		Kind:     FunctionNCM,
		Instance: "usb2",
		Net:      &NetFunction{CompatibleID: "WINNCM"},
	})

	if created := requestJSON(t, router, http.MethodPost, "/profiles", profile); created.Code != 0 {
		t.Fatalf("create = %+v", created)
	}
	before := len(ops.Trace())
	preview := decodeData[Preview](t, requestJSON(t, router, http.MethodPut, "/preview", profile))
	if preview.Valid || len(preview.Errors) == 0 {
		t.Fatalf("preview = %+v", preview)
	}
	if applied := requestJSON(t, router, http.MethodPut, "/apply", nameRequest{Name: profile.Name}); applied.Code == 0 {
		t.Fatal("incompatible profile applied")
	}
	if after := len(ops.Trace()); after != before {
		t.Fatalf("failed apply emitted %d ops", after-before)
	}
}

func TestJSONRejectsDuplicateKeys(t *testing.T) {
	service, _ := testPresentationService(t)
	router := profileRouter(service)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/profiles/standard/clone", bytes.NewBufferString(`{"name":"one","name":"two"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	var envelope responseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code == 0 {
		t.Fatal("duplicate key accepted")
	}
}

func requestJSON(t *testing.T, handler http.Handler, method, path string, body any) responseEnvelope {
	t.Helper()
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)
	var envelope responseEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode %s %s: %v body=%s", method, path, err, recorder.Body.String())
	}
	return envelope
}

func decodeData[T any](t *testing.T, envelope responseEnvelope) T {
	t.Helper()
	var value T
	if envelope.Code != 0 {
		t.Fatalf("response failed: %+v", envelope)
	}
	if err := json.Unmarshal(envelope.Data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

// The apply dialog had one fixed sentence to describe every change. The preview
// carries what the apply takes away, what it leaves the operator holding and
// where a failure lands, all of it before anything is written.
func TestPreviewSaysWhatTheApplyWillDo(t *testing.T) {
	service, _ := testPresentationService(t)
	router := profileRouter(service)

	rich := profileForFlags(flags{rndis: true, disk: true})
	rich.Name, rich.BuiltIn = "rich", false
	if created := requestJSON(t, router, http.MethodPost, "/profiles", rich); created.Code != 0 {
		t.Fatalf("create = %+v", created)
	}
	if applied := requestJSON(t, router, http.MethodPut, "/apply", nameRequest{Name: rich.Name}); applied.Code != 0 {
		t.Fatalf("apply = %+v", applied)
	}

	preview := decodeData[Preview](t, requestJSON(t, router, http.MethodPut, "/preview", standardProfile()))
	if !preview.Valid {
		t.Fatalf("preview = %+v", preview)
	}
	if preview.Apply == nil || preview.Rollback == nil {
		t.Fatalf("preview says nothing about the apply: %+v", preview)
	}

	want := []string{"rndis.usb0", "mass_storage.disk0"}
	if strings.Join(preview.Apply.Removes, ",") != strings.Join(want, ",") {
		t.Fatalf("removes = %v, want %v", preview.Apply.Removes, want)
	}
	if !preview.Apply.HID || preview.Apply.Recovery != RecoveryReboot {
		t.Fatalf("apply = %+v, want hid kept and a host reboot", preview.Apply)
	}
	if preview.Rollback.Profile != rich.Name || len(preview.Rollback.Removes) != 0 {
		t.Fatalf("rollback = %+v, want %s restored with nothing removed", preview.Rollback, rich.Name)
	}
	if preview.Device.ProductID != standardProfile().Device.ProductID || preview.Device.Serial == "" {
		t.Fatalf("device = %+v, want the resolved identity", preview.Device)
	}
	if len(preview.FIFOs) == 0 {
		t.Fatalf("fifos = %v, want the seating the compiler already did", preview.FIFOs)
	}
}

// An apply that binds and verifies is still an apply the attached host can hate,
// and until now the only rollback was the one a failed apply ran for itself.
func TestRollbackReturnsToTheProfileBeforeTheActiveOne(t *testing.T) {
	service, ops := testPresentationService(t)
	router := profileRouter(service)

	desk := profileForFlags(flags{rndis: true, disk: true})
	desk.Name, desk.BuiltIn = "desk", false
	if created := requestJSON(t, router, http.MethodPost, "/profiles", desk); created.Code != 0 {
		t.Fatalf("create = %+v", created)
	}
	if applied := requestJSON(t, router, http.MethodPut, "/apply", nameRequest{Name: ProfileHIDOnly}); applied.Code != 0 {
		t.Fatalf("apply hid-only = %+v", applied)
	}
	if applied := requestJSON(t, router, http.MethodPut, "/apply", nameRequest{Name: desk.Name}); applied.Code != 0 {
		t.Fatalf("apply desk = %+v", applied)
	}

	status := decodeData[Status](t, requestJSON(t, router, http.MethodGet, "/status", nil))
	if status.LastKnownGood != desk.Name || status.RollbackTarget != ProfileHIDOnly {
		t.Fatalf("status = %+v, want %s good with %s behind it", status, desk.Name, ProfileHIDOnly)
	}

	rolled := decodeData[struct {
		Profile string `json:"profile"`
	}](t, requestJSON(t, router, http.MethodPost, "/rollback", nil))
	if rolled.Profile != ProfileHIDOnly {
		t.Fatalf("rollback = %+v, want %s", rolled, ProfileHIDOnly)
	}
	if links := ops.Links(); links[configPrefix+"/rndis.usb0"] != "" {
		t.Fatalf("rollback left the network function linked: %v", links)
	}
	status = decodeData[Status](t, requestJSON(t, router, http.MethodGet, "/status", nil))
	if status.Snapshot.Active != ProfileHIDOnly {
		t.Fatalf("active = %q, want %s", status.Snapshot.Active, ProfileHIDOnly)
	}

	// The second click of a double click reports the same landing and does not
	// unbind the gadget a second time.
	before := len(ops.Trace())
	again := decodeData[struct {
		Profile string `json:"profile"`
	}](t, requestJSON(t, router, http.MethodPost, "/rollback", nil))
	if again.Profile != ProfileHIDOnly {
		t.Fatalf("second rollback = %+v, want %s", again, ProfileHIDOnly)
	}
	if after := len(ops.Trace()); after != before {
		t.Fatalf("second rollback emitted %d ops", after-before)
	}
}

func TestRollbackRefusesWithNothingToRollBackTo(t *testing.T) {
	service, ops := testPresentationService(t)
	router := profileRouter(service)
	if applied := requestJSON(t, router, http.MethodPut, "/apply", nameRequest{Name: ProfileStandard}); applied.Code != 0 {
		t.Fatalf("apply = %+v", applied)
	}

	before := len(ops.Trace())
	refused := requestJSON(t, router, http.MethodPost, "/rollback", nil)
	if refused.Code == 0 {
		t.Fatalf("rollback = %+v, want a refusal", refused)
	}
	if !strings.Contains(refused.Msg, "no earlier profile") {
		t.Fatalf("refusal = %q", refused.Msg)
	}
	if after := len(ops.Trace()); after != before {
		t.Fatalf("refused rollback emitted %d ops", after-before)
	}
}

// usb-proxy holds the same udc->driver pointer, so a rollback is a gadget
// mutation like any other and the loan refuses it.
func TestRollbackIsRefusedWhileTheUDCIsOnLoan(t *testing.T) {
	service, ops := testPresentationService(t)
	router := profileRouter(service)
	for _, name := range []string{ProfileHIDOnly, ProfileStandard} {
		if applied := requestJSON(t, router, http.MethodPut, "/apply", nameRequest{Name: name}); applied.Code != 0 {
			t.Fatalf("apply %s = %+v", name, applied)
		}
	}
	if _, err := service.manager.SurrenderUDC(); err != nil {
		t.Fatalf("surrender: %v", err)
	}

	refused := requestJSON(t, router, http.MethodPost, "/rollback", nil)
	if refused.Code == 0 || !strings.Contains(refused.Msg, "usb-proxy") {
		t.Fatalf("rollback during a passthrough session = %+v", refused)
	}
	if bound := ops.Bound(); bound != "" {
		t.Fatalf("rollback bound the udc to %q while usb-proxy had it", bound)
	}
}
