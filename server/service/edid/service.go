package edid

import (
	"context"
	"encoding/base64"
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

// A valid EDID is 256 bytes. The slack lets the validator reject an oversized
// file by its real size rather than a truncating read doing it silently.
const uploadLimit = 64 << 10

type Service struct {
	store   *Store
	applier *Applier
}

func NewService(capture CaptureGuard) *Service {
	return &Service{store: NewStore(), applier: NewApplier(capture)}
}

// Reports what NanoKVM last flashed and verified, not what is on the chip:
// there is no chip read, so an empty archive answers unverifiedSinceBoot.
func (s *Service) GetEdid(c *gin.Context) {
	var rsp proto.Response

	data, record, err := s.store.LoadActive()
	if err != nil {
		log.Errorf("edid: read store failed: %s", err)
		rsp.ErrRsp(c, -1, "read edid store failed")
		return
	}

	pre := Preflight()
	result := &proto.GetEdidRsp{
		UnverifiedSinceBoot: data == nil,
		FactoryAvailable:    FactoryAvailable(),
		Preflight: proto.EdidPreflight{
			Chip:               string(pre.Chip),
			Product:            string(pre.Product),
			Supported:          pre.Supported,
			RequiresPowerCycle: pre.RequiresPowerCycle,
			ToolAvailable:      pre.ToolAvailable,
			Reason:             pre.Reason,
		},
		Backups: []proto.EdidBackup{},
	}

	if data != nil {
		if decoded, err := Decode(data); err == nil {
			summary := summarize(decoded, data)
			result.Active = &summary
		} else {
			log.Errorf("edid: archived blob no longer decodes: %s", err)
		}
	}
	if record != nil {
		result.Source = record.Source
		result.AppliedAt = record.AppliedAt.UTC().Format(time.RFC3339)
	}

	backups, err := s.store.History()
	if err != nil {
		log.Errorf("edid: read history failed: %s", err)
	}
	for _, backup := range backups {
		result.Backups = append(result.Backups, proto.EdidBackup{
			ID:        backup.ID,
			SHA256:    backup.SHA256,
			AppliedAt: backup.AppliedAt.UTC().Format(time.RFC3339),
			Size:      backup.Size,
		})
	}

	rsp.OkRspWithData(c, result)
}

// Summaries only. The bytes travel back through apply, keyed by id.
func (s *Service) GetProfiles(c *gin.Context) {
	var rsp proto.Response

	shipped := Profiles()
	result := &proto.GetEdidProfilesRsp{Profiles: make([]proto.EdidProfile, 0, len(shipped))}
	for _, profile := range shipped {
		result.Profiles = append(result.Profiles, proto.EdidProfile{
			ID:            profile.ID(),
			Manufacturer:  profile.Manufacturer,
			Model:         profile.Model,
			PreferredMode: profile.PreferredMode,
			Source:        profile.Source,
		})
	}

	rsp.OkRspWithData(c, result)
}

// Touches no lock, no chip and no store.
func (s *Service) DecodeEdid(c *gin.Context) {
	var rsp proto.Response

	blob, err := uploadedBlob(c)
	if err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}

	normalized, err := Normalize(blob)
	if err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	decoded, err := Decode(normalized)
	if err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	rsp.OkRspWithData(c, &proto.DecodeEdidRsp{
		Summary: summarize(decoded, normalized),
		Detail:  decoded,
	})
}

func (s *Service) ApplyEdid(c *gin.Context) {
	var req proto.ApplyEdidReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	blob, source, err := s.selectBlob(req)
	if err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	s.flash(c, blob, source)
}

func (s *Service) selectBlob(req proto.ApplyEdidReq) ([]byte, string, error) {
	switch {
	case req.Profile != "" && req.Data != "":
		return nil, "", errors.New("pass either a profile or a blob, not both")

	case req.Profile != "":
		profile, err := ProfileByID(req.Profile)
		if err != nil {
			return nil, "", err
		}
		return profile.Data, "profile:" + profile.ID(), nil

	case req.Data != "":
		blob, err := decodeBase64(req.Data)
		if err != nil {
			return nil, "", err
		}
		return blob, "upload", nil

	default:
		return nil, "", errors.New("no edid supplied")
	}
}

// Re-flashing an archived file is the only rollback primitive that exists: the
// tool has no read mode, so there is no pre-image.
func (s *Service) RestoreEdid(c *gin.Context) {
	var req proto.RestoreEdidReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	blob, source, err := s.restoreTarget(req)
	if err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	s.flash(c, blob, source)
}

func (s *Service) restoreTarget(req proto.RestoreEdidReq) ([]byte, string, error) {
	if req.Source == "factory" {
		blob, err := FactoryImage()
		if err != nil {
			return nil, "", err
		}
		return blob, "factory", nil
	}

	id := req.ID
	if id == "" {
		backups, err := s.store.History()
		if err != nil {
			return nil, "", err
		}
		if len(backups) == 0 {
			return nil, "", errors.New("no backup to restore")
		}
		id = backups[0].ID
	}

	blob, err := s.store.ReadBackup(id)
	if err != nil {
		return nil, "", err
	}
	return blob, "history:" + id, nil
}

func (s *Service) flash(c *gin.Context, blob []byte, source string) {
	var rsp proto.Response

	// A browser navigating away mid-write must not cancel the child and leave
	// the region half written.
	ctx := context.WithoutCancel(c.Request.Context())

	outcome, err := s.applier.Apply(ctx, blob, source)
	if err != nil {
		if errors.Is(err, ErrLocked) {
			rsp.ErrRsp(c, -3, err.Error())
			return
		}
		log.Errorf("edid: apply failed: %s", err)
		rsp.ErrRsp(c, -4, "apply failed")
		return
	}

	result := &proto.ApplyEdidRsp{
		State:              outcome.State,
		Verified:           outcome.Verified,
		Retryable:          outcome.Retryable,
		RequiresPowerCycle: outcome.RequiresPowerCycle,
		Message:            outcome.Message,
		WrittenHex:         outcome.WrittenHex,
		ReadHex:            outcome.ReadHex,
	}
	if outcome.Verified {
		if decoded, err := Decode(blob); err == nil {
			summary := summarize(decoded, blob)
			result.Summary = &summary
		}
	}

	rsp.OkRspWithData(c, result)
}

func (s *Service) DownloadEdid(c *gin.Context) {
	var rsp proto.Response

	data, _, err := s.store.LoadActive()
	if err != nil {
		log.Errorf("edid: read store failed: %s", err)
		rsp.ErrRsp(c, -1, "read edid store failed")
		return
	}
	if data == nil {
		rsp.ErrRsp(c, -2, "no edid has been applied by this device")
		return
	}

	serveBlob(c, "nanokvm-edid-"+digest(data)[:8]+".bin", data)
}

func (s *Service) DownloadBackup(c *gin.Context) {
	var req proto.GetEdidBackupReq
	var rsp proto.Response

	if err := proto.ParseQueryRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	data, err := s.store.ReadBackup(req.ID)
	if err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	serveBlob(c, "nanokvm-edid-"+req.ID+".bin", data)
}

func serveBlob(c *gin.Context, name string, data []byte) {
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// Either a multipart file or a base64 body.
func uploadedBlob(c *gin.Context) ([]byte, error) {
	if file, _, err := c.Request.FormFile("file"); err == nil {
		defer func() { _ = file.Close() }()

		blob, err := io.ReadAll(io.LimitReader(file, uploadLimit))
		if err != nil {
			return nil, errors.New("read uploaded file failed")
		}
		return blob, nil
	}

	var req proto.DecodeEdidReq
	if err := proto.ParseFormRequest(c, &req); err != nil {
		return nil, errors.New("invalid arguments")
	}
	if req.Data == "" {
		return nil, errors.New("no edid supplied")
	}
	return decodeBase64(req.Data)
}

func decodeBase64(encoded string) ([]byte, error) {
	blob, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, errors.New("edid data is not valid base64")
	}
	return blob, nil
}

func summarize(decoded *EDID, blob []byte) proto.EdidSummary {
	summary := proto.EdidSummary{
		SHA256:       digest(blob),
		Manufacturer: decoded.Manufacturer,
		Model:        decoded.Name(),
		ProductCode:  decoded.ProductCode,
		Serial:       decoded.Serial,
		Week:         decoded.Week,
		Year:         decoded.Year,
		Version:      fmt.Sprintf("%d.%d", decoded.Version, decoded.Revision),
		Extensions:   decoded.Extensions,
		Audio:        HasAudio(decoded),
	}

	if timing := decoded.PreferredTiming(); timing != nil {
		summary.PreferredMode = timing.Mode()
		summary.PixelClockKHz = timing.PixelClockKHz
	}
	return summary
}
