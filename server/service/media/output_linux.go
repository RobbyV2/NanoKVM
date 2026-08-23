//go:build linux && cgo

package media

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <errno.h>
#include <fcntl.h>
#include <linux/usb/g_uvc.h>
#include <linux/usb/video.h>
#include <linux/videodev2.h>
#include <poll.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <unistd.h>

#define NK_BUFFERS 4
#define NK_FRAMES 8

_Static_assert(sizeof(struct usb_ctrlrequest) == 8, "usb_ctrlrequest ABI");
_Static_assert(sizeof(struct uvc_request_data) == 64, "uvc_request_data ABI");
_Static_assert(sizeof(struct uvc_event) == 64, "uvc_event ABI");
_Static_assert(sizeof(struct uvc_streaming_control) == 34, "uvc_streaming_control ABI");
_Static_assert(sizeof(struct v4l2_event) == 136, "v4l2_event ABI");

struct nk_frame {
	uint16_t width;
	uint16_t height;
	uint32_t interval;
};

struct nk_uvc {
	int fd;
	int streaming;
	int queued;
	int pending;
	unsigned int count;
	void *maps[NK_BUFFERS];
	size_t lengths[NK_BUFFERS];
	struct nk_frame frames[NK_FRAMES];
	unsigned int frame_count;
	uint32_t max_packet;
	struct uvc_streaming_control probe;
	struct uvc_streaming_control commit;
};

static int nk_ioctl(int fd, unsigned long request, void *arg) {
	int rc;
	do { rc = ioctl(fd, request, arg); } while (rc < 0 && errno == EINTR);
	return rc;
}

static void nk_control(struct nk_uvc *u, struct uvc_streaming_control *c, unsigned int frame) {
	if (frame >= u->frame_count) frame = 0;
	memset(c, 0, sizeof(*c));
	c->bmHint = 1;
	c->bFormatIndex = 1;
	c->bFrameIndex = frame + 1;
	c->dwFrameInterval = u->frames[frame].interval;
	c->dwMaxVideoFrameSize = u->frames[frame].width * u->frames[frame].height * 2;
	c->dwMaxPayloadTransferSize = u->max_packet;
}

static int nk_uvc_format(struct nk_uvc *u, unsigned int frame) {
	if (frame >= u->frame_count) return -EINVAL;
	struct v4l2_format format;
	memset(&format, 0, sizeof(format));
	format.type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
	format.fmt.pix.width = u->frames[frame].width;
	format.fmt.pix.height = u->frames[frame].height;
	format.fmt.pix.pixelformat = V4L2_PIX_FMT_MJPEG;
	format.fmt.pix.field = V4L2_FIELD_NONE;
	format.fmt.pix.sizeimage = u->frames[frame].width * u->frames[frame].height * 2;
	return nk_ioctl(u->fd, VIDIOC_S_FMT, &format);
}

static void nk_uvc_close(struct nk_uvc *u);
static int nk_uvc_queue(struct nk_uvc *u, const void *data, size_t length, int all);

static struct nk_uvc *nk_uvc_open(const char *path, const uint16_t *widths,
	const uint16_t *heights, const uint32_t *intervals, unsigned int frame_count,
	uint32_t max_packet) {
	if (frame_count == 0 || frame_count > NK_FRAMES) { errno = EINVAL; return NULL; }
	struct nk_uvc *u = calloc(1, sizeof(*u));
	if (!u) return NULL;
	u->fd = open(path, O_RDWR | O_NONBLOCK | O_CLOEXEC);
	if (u->fd < 0) { free(u); return NULL; }
	u->frame_count = frame_count;
	u->max_packet = max_packet;
	for (unsigned int i = 0; i < frame_count; i++) {
		u->frames[i].width = widths[i];
		u->frames[i].height = heights[i];
		u->frames[i].interval = intervals[i];
	}
	nk_control(u, &u->probe, 0);
	nk_control(u, &u->commit, 0);

	for (unsigned int type = UVC_EVENT_CONNECT; type <= UVC_EVENT_DATA; type++) {
		struct v4l2_event_subscription subscription;
		memset(&subscription, 0, sizeof(subscription));
		subscription.type = type;
		if (nk_ioctl(u->fd, VIDIOC_SUBSCRIBE_EVENT, &subscription) < 0) goto fail;
	}
	return u;
fail:
	nk_uvc_close(u);
	return NULL;
}

static void nk_uvc_release_buffers(struct nk_uvc *u) {
	for (unsigned int i = 0; i < u->count; i++) {
		if (u->maps[i]) munmap(u->maps[i], u->lengths[i]);
		u->maps[i] = NULL;
		u->lengths[i] = 0;
	}
	u->count = 0;
	u->queued = 0;
	struct v4l2_requestbuffers request;
	memset(&request, 0, sizeof(request));
	request.type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
	request.memory = V4L2_MEMORY_MMAP;
	nk_ioctl(u->fd, VIDIOC_REQBUFS, &request);
}

static int nk_uvc_start(struct nk_uvc *u, const void *data, size_t length) {
	unsigned int frame = u->commit.bFrameIndex > 0 ? u->commit.bFrameIndex - 1 : 0;
	if (frame >= u->frame_count) frame = 0;
	nk_uvc_release_buffers(u);
	if (nk_uvc_format(u, frame) < 0) return -errno;
	struct v4l2_requestbuffers request;
	memset(&request, 0, sizeof(request));
	request.count = NK_BUFFERS;
	request.type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
	request.memory = V4L2_MEMORY_MMAP;
	if (nk_ioctl(u->fd, VIDIOC_REQBUFS, &request) < 0) return -errno;
	if (request.count == 0) return -ENOBUFS;
	u->count = request.count > NK_BUFFERS ? NK_BUFFERS : request.count;
	for (unsigned int i = 0; i < u->count; i++) {
		struct v4l2_buffer buffer;
		memset(&buffer, 0, sizeof(buffer));
		buffer.type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
		buffer.memory = V4L2_MEMORY_MMAP;
		buffer.index = i;
		if (nk_ioctl(u->fd, VIDIOC_QUERYBUF, &buffer) < 0) return -errno;
		u->lengths[i] = buffer.length;
		u->maps[i] = mmap(NULL, buffer.length, PROT_READ | PROT_WRITE, MAP_SHARED, u->fd, buffer.m.offset);
		if (u->maps[i] == MAP_FAILED) { u->maps[i] = NULL; return -errno; }
	}
	int rc = nk_uvc_queue(u, data, length, 1);
	if (rc < 0) return rc;
	enum v4l2_buf_type type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
	if (nk_ioctl(u->fd, VIDIOC_STREAMON, &type) < 0) return -errno;
	u->streaming = 1;
	return 0;
}

static int nk_uvc_restart(struct nk_uvc *u, const void *data, size_t length) {
	if (!u->streaming) return 0;
	enum v4l2_buf_type type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
	if (nk_ioctl(u->fd, VIDIOC_STREAMOFF, &type) < 0) return -errno;
	u->streaming = 0;
	u->queued = 0;
	return nk_uvc_start(u, data, length);
}

static int nk_uvc_queue(struct nk_uvc *u, const void *data, size_t length, int all) {
	unsigned int begin = 0, end = all ? u->count : 1;
	for (unsigned int i = begin; i < end; i++) {
		if (length > u->lengths[i]) return -EMSGSIZE;
		memcpy(u->maps[i], data, length);
		struct v4l2_buffer buffer;
		memset(&buffer, 0, sizeof(buffer));
		buffer.type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
		buffer.memory = V4L2_MEMORY_MMAP;
		buffer.index = i;
		buffer.bytesused = length;
		if (nk_ioctl(u->fd, VIDIOC_QBUF, &buffer) < 0) return -errno;
	}
	if (all) u->queued = 1;
	return 0;
}

static int nk_uvc_response(struct nk_uvc *u, struct uvc_request_data *response) {
	return nk_ioctl(u->fd, UVCIOC_SEND_RESPONSE, response) < 0 ? -errno : 0;
}

static int nk_uvc_setup(struct nk_uvc *u, const struct usb_ctrlrequest *request) {
	struct uvc_request_data response;
	memset(&response, 0, sizeof(response));
	response.length = -EL2HLT;
	u->pending = 0;
	if ((request->bRequestType & (USB_TYPE_MASK | USB_RECIP_MASK)) !=
	    (USB_TYPE_CLASS | USB_RECIP_INTERFACE) || (request->wIndex >> 8) != 0)
		return nk_uvc_response(u, &response);
	uint8_t selector = request->wValue >> 8;
	if (selector != UVC_VS_PROBE_CONTROL && selector != UVC_VS_COMMIT_CONTROL)
		return nk_uvc_response(u, &response);
	struct uvc_streaming_control *control = selector == UVC_VS_PROBE_CONTROL ? &u->probe : &u->commit;
	unsigned int length = request->wLength;
	if (length > sizeof(*control)) length = sizeof(*control);
	switch (request->bRequest) {
	case UVC_SET_CUR:
		u->pending = selector;
		response.length = length;
		break;
	case UVC_GET_CUR:
		response.length = length;
		memcpy(response.data, control, length);
		break;
	case UVC_GET_MIN:
	case UVC_GET_DEF:
		nk_control(u, (struct uvc_streaming_control *)response.data, 0);
		response.length = length;
		break;
	case UVC_GET_MAX:
		nk_control(u, (struct uvc_streaming_control *)response.data, u->frame_count - 1);
		response.length = length;
		break;
	case UVC_GET_RES:
		response.length = length;
		break;
	case UVC_GET_LEN:
		response.data[0] = sizeof(*control) & 0xff;
		response.data[1] = sizeof(*control) >> 8;
		response.length = 2;
		break;
	case UVC_GET_INFO:
		response.data[0] = 3;
		response.length = 1;
		break;
	}
	return nk_uvc_response(u, &response);
}

static int nk_uvc_data(struct nk_uvc *u, const struct uvc_request_data *data) {
	struct uvc_streaming_control *control = u->pending == UVC_VS_COMMIT_CONTROL ? &u->commit : &u->probe;
	if (data->length <= 0) return 0;
	size_t length = data->length > sizeof(*control) ? sizeof(*control) : data->length;
	memcpy(control, data->data, length);
	unsigned int frame = control->bFrameIndex > 0 ? control->bFrameIndex - 1 : 0;
	if (frame >= u->frame_count) frame = 0;
	uint32_t requested = control->dwFrameInterval;
	nk_control(u, control, frame);
	if (requested == 333333 || requested == 666666) control->dwFrameInterval = requested;
	if (u->pending == UVC_VS_COMMIT_CONTROL && !u->streaming && nk_uvc_format(u, frame) < 0)
		return -errno;
	u->pending = 0;
	return 0;
}

static int nk_uvc_step(struct nk_uvc *u, const void *data, size_t length,
	int timeout_ms, unsigned int *width, unsigned int *height, unsigned int *fps) {
	struct pollfd descriptor = { .fd = u->fd, .events = POLLPRI | (u->streaming ? POLLOUT : 0) };
	int polled = poll(&descriptor, 1, timeout_ms);
	if (polled < 0) return errno == EINTR ? 0 : -errno;
	int edge = 0;
	if (descriptor.revents & POLLPRI) {
		for (unsigned int handled = 0; handled < 64; handled++) {
			struct v4l2_event event;
			memset(&event, 0, sizeof(event));
			if (nk_ioctl(u->fd, VIDIOC_DQEVENT, &event) < 0) {
				if (errno == EAGAIN) break;
				return -errno;
			}
			struct uvc_event *uvc = (struct uvc_event *)event.u.data;
			int rc = 0;
			switch (event.type) {
			case UVC_EVENT_SETUP: rc = nk_uvc_setup(u, &uvc->req); break;
			case UVC_EVENT_DATA: rc = nk_uvc_data(u, &uvc->data); break;
			case UVC_EVENT_STREAMON:
				if (!u->streaming) edge = 3;
				break;
			case UVC_EVENT_STREAMOFF:
			case UVC_EVENT_DISCONNECT:
				if (u->streaming) {
					enum v4l2_buf_type type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
					nk_ioctl(u->fd, VIDIOC_STREAMOFF, &type);
				}
				u->streaming = 0; u->queued = 0; edge = 2;
				break;
			}
			if (rc < 0) return rc;
		}
	}
	if (descriptor.revents & (POLLHUP | POLLNVAL)) return -EIO;
	if (u->streaming && (descriptor.revents & POLLERR)) return -EIO;
	unsigned int frame = u->commit.bFrameIndex > 0 ? u->commit.bFrameIndex - 1 : 0;
	if (frame >= u->frame_count) frame = 0;
	*width = u->frames[frame].width;
	*height = u->frames[frame].height;
	*fps = u->commit.dwFrameInterval ? 10000000 / u->commit.dwFrameInterval : 0;
	if (!u->streaming || !(descriptor.revents & POLLOUT)) return edge;
	struct v4l2_buffer buffer;
	memset(&buffer, 0, sizeof(buffer));
	buffer.type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
	buffer.memory = V4L2_MEMORY_MMAP;
	if (nk_ioctl(u->fd, VIDIOC_DQBUF, &buffer) < 0) return errno == EAGAIN ? edge : -errno;
	if (buffer.index >= u->count) return -EIO;
	if (length > u->lengths[buffer.index]) return -EMSGSIZE;
	memcpy(u->maps[buffer.index], data, length);
	buffer.bytesused = length;
	if (nk_ioctl(u->fd, VIDIOC_QBUF, &buffer) < 0) return -errno;
	return edge;
}

static void nk_uvc_close(struct nk_uvc *u) {
	if (!u) return;
	if (u->fd >= 0 && u->streaming) {
		enum v4l2_buf_type type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
		nk_ioctl(u->fd, VIDIOC_STREAMOFF, &type);
	}
	nk_uvc_release_buffers(u);
	if (u->fd >= 0) close(u->fd);
	free(u);
}

enum nk_pcm_format { NK_PCM_S16_LE = 0 };
enum nk_pcm_flags { NK_PCM_OUT = 0 };
struct nk_pcm_config {
	unsigned int channels, rate, period_size, period_count;
	int format;
	unsigned int start_threshold, stop_threshold, silence_threshold, silence_size;
	int avail_min;
};
_Static_assert(sizeof(struct nk_pcm_config) == 40, "packaged tinyalsa pcm_config ABI");
struct nk_pcm {
	void *library;
	void *handle;
	int (*write)(void *, const void *, unsigned int);
	int (*wait)(void *, int);
	int (*prepare)(void *);
	int (*stop)(void *);
	int (*close)(void *);
};

static struct nk_pcm *nk_pcm_open(unsigned int card, unsigned int device) {
	const char *paths[] = { "/tmp/server/dl_lib/libtinyalsa.so", "/kvmapp/server/dl_lib/libtinyalsa.so", "libtinyalsa.so" };
	void *library = NULL;
	for (unsigned int i = 0; i < sizeof(paths) / sizeof(paths[0]); i++) {
		library = dlopen(paths[i], RTLD_NOW | RTLD_LOCAL);
		if (library) break;
	}
	if (!library) { errno = ENOENT; return NULL; }
	void *(*open_pcm)(unsigned int, unsigned int, unsigned int, const struct nk_pcm_config *) = dlsym(library, "pcm_open");
	int (*is_ready)(void *) = dlsym(library, "pcm_is_ready");
	int (*write_pcm)(void *, const void *, unsigned int) = dlsym(library, "pcm_write");
	int (*wait_pcm)(void *, int) = dlsym(library, "pcm_wait");
	int (*prepare_pcm)(void *) = dlsym(library, "pcm_prepare");
	int (*stop_pcm)(void *) = dlsym(library, "pcm_stop");
	int (*close_pcm)(void *) = dlsym(library, "pcm_close");
	if (!open_pcm || !is_ready || !write_pcm || !wait_pcm || !prepare_pcm || !stop_pcm || !close_pcm) { dlclose(library); errno = ENOSYS; return NULL; }
	struct nk_pcm_config config = { .channels = 1, .rate = 48000, .period_size = 960,
		.period_count = 4, .format = NK_PCM_S16_LE, .avail_min = 960 };
	void *handle = open_pcm(card, device, NK_PCM_OUT, &config);
	if (!handle || !is_ready(handle)) { if (handle) close_pcm(handle); dlclose(library); errno = ENODEV; return NULL; }
	struct nk_pcm *pcm = calloc(1, sizeof(*pcm));
	if (!pcm) { close_pcm(handle); dlclose(library); return NULL; }
	pcm->library = library; pcm->handle = handle; pcm->write = write_pcm; pcm->wait = wait_pcm;
	pcm->prepare = prepare_pcm; pcm->stop = stop_pcm; pcm->close = close_pcm;
	return pcm;
}

static int nk_pcm_write(struct nk_pcm *pcm, const void *data, unsigned int length) {
	int ready = pcm->wait(pcm->handle, 0);
	if (ready == 0) return -EAGAIN;
	if (ready < 0) return ready;
	int rc = pcm->write(pcm->handle, data, length);
	return rc < 0 ? rc : 0;
}

static int nk_pcm_reset(struct nk_pcm *pcm) {
	int rc = pcm->stop(pcm->handle);
	if (rc < 0) return rc;
	return pcm->prepare(pcm->handle);
}

static void nk_pcm_close(struct nk_pcm *pcm) {
	if (!pcm) return;
	pcm->close(pcm->handle);
	dlclose(pcm->library);
	free(pcm);
}

static int nk_errno(void) { return errno; }
*/
import "C"

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"NanoKVM-Server/service/sources"
)

type platformFactory struct{}

func (platformFactory) Open(spec SlotSpec, node string) (Output, error) {
	if spec.Kind == sources.KindCamera {
		return openUVC(spec, node)
	}
	return openPCM(node)
}

type uvcOutput struct {
	mu     sync.Mutex
	handle *C.struct_nk_uvc
}

func openUVC(spec SlotSpec, node string) (*uvcOutput, error) {
	var widths [8]C.uint16_t
	var heights [8]C.uint16_t
	var intervals [8]C.uint32_t
	frames := spec.Video.Formats[0].Frames
	if len(frames) > len(widths) {
		return nil, fmt.Errorf("too many video frames: %d", len(frames))
	}
	for i, frame := range frames {
		widths[i], heights[i] = C.uint16_t(frame.Width), C.uint16_t(frame.Height)
		intervals[i] = C.uint32_t(frame.Intervals[0])
	}
	path := C.CString(node)
	defer C.free(unsafe.Pointer(path))
	handle := C.nk_uvc_open(path, &widths[0], &heights[0], &intervals[0], C.uint(len(frames)), C.uint32_t(spec.Video.StreamingMaxPacket))
	if handle == nil {
		return nil, fmt.Errorf("open UVC %s: %w", node, errno())
	}
	return &uvcOutput{handle: handle}, nil
}

func (o *uvcOutput) Run(ctx context.Context, frames <-chan Packet, fallback Fallback, demand func(sources.Demand), source func(bool)) error {
	current, err := fallback(0, 0)
	if err != nil {
		return err
	}
	var width, height, fps C.uint
	var generation uint64
	var sourceAt time.Time
	sourceActive := false
	for {
		select {
		case <-ctx.Done():
			return nil
		case frame := <-frames:
			changed := frame.Generation != generation
			if frame.Reset {
				current, err = fallback(int(width), int(height))
				if err != nil {
					return err
				}
				if sourceActive {
					sourceActive = false
					source(false)
				}
			} else {
				current = frame
				sourceAt = time.Now()
				if !sourceActive {
					sourceActive = true
					source(true)
				}
			}
			if changed {
				generation = frame.Generation
				o.mu.Lock()
				if o.handle == nil {
					o.mu.Unlock()
					return nil
				}
				rc := C.nk_uvc_restart(o.handle, unsafe.Pointer(&current.Data[0]), C.size_t(len(current.Data)))
				o.mu.Unlock()
				if rc < 0 {
					return fmt.Errorf("reset UVC: %s", syscallError(rc))
				}
			}
		default:
		}
		if sourceActive && time.Since(sourceAt) >= videoFallback {
			current, err = fallback(int(width), int(height))
			if err != nil {
				return err
			}
			sourceActive = false
			source(false)
		}
		o.mu.Lock()
		if o.handle == nil {
			o.mu.Unlock()
			return nil
		}
		rc := C.nk_uvc_step(o.handle, unsafe.Pointer(&current.Data[0]), C.size_t(len(current.Data)), 25, &width, &height, &fps)
		o.mu.Unlock()
		if rc < 0 {
			return fmt.Errorf("write UVC: %s", syscallError(rc))
		}
		if rc == 3 {
			current, err = fallback(int(width), int(height))
			if err != nil {
				return err
			}
			if sourceActive {
				sourceActive = false
				source(false)
			}
			o.mu.Lock()
			if o.handle == nil {
				o.mu.Unlock()
				return nil
			}
			rc = C.nk_uvc_start(o.handle, unsafe.Pointer(&current.Data[0]), C.size_t(len(current.Data)))
			o.mu.Unlock()
			if rc < 0 {
				return fmt.Errorf("start UVC: %s", syscallError(rc))
			}
			demand(sources.Demand{Streaming: true, Width: int(width), Height: int(height), FPS: int(fps), Since: time.Now().UTC()})
		} else if rc == 2 {
			demand(sources.Demand{})
			if sourceActive {
				sourceActive = false
				source(false)
			}
		}
	}
}

func (o *uvcOutput) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.handle != nil {
		C.nk_uvc_close(o.handle)
		o.handle = nil
	}
	return nil
}

type pcmOutput struct {
	mu     sync.Mutex
	handle *C.struct_nk_pcm
}

func openPCM(node string) (*pcmOutput, error) {
	card, device, err := parseALSANode(node)
	if err != nil {
		return nil, err
	}
	handle := C.nk_pcm_open(C.uint(card), C.uint(device))
	if handle == nil {
		return nil, fmt.Errorf("open PCM %s: %w", node, errno())
	}
	return &pcmOutput{handle: handle}, nil
}

func (o *pcmOutput) Run(ctx context.Context, frames <-chan Packet, fallback Fallback, demand func(sources.Demand), source func(bool)) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	current, err := fallback(0, 0)
	if err != nil {
		return err
	}
	streaming, failures, successes := false, 0, 0
	sourceActive := false
	currentSource := false
	var generation uint64
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			select {
			case frame := <-frames:
				changed := frame.Generation != generation
				if frame.Reset {
					current, err = fallback(0, 0)
					if err != nil {
						return err
					}
					currentSource = false
				} else {
					current = frame
					currentSource = true
				}
				if changed {
					generation = frame.Generation
					o.mu.Lock()
					if o.handle == nil {
						o.mu.Unlock()
						return nil
					}
					rc := C.nk_pcm_reset(o.handle)
					o.mu.Unlock()
					if rc < 0 {
						return fmt.Errorf("reset PCM: return %d", int(rc))
					}
					streaming, failures, successes = false, 0, 0
					demand(sources.Demand{})
				}
			default:
			}
			o.mu.Lock()
			if o.handle == nil {
				o.mu.Unlock()
				return nil
			}
			rc := C.nk_pcm_write(o.handle, unsafe.Pointer(&current.Data[0]), C.uint(len(current.Data)))
			o.mu.Unlock()
			if rc == 0 {
				failures = 0
				successes++
				if !streaming && successes > 4 {
					streaming = true
					demand(sources.Demand{Streaming: true, Since: time.Now().UTC()})
				}
				if currentSource != sourceActive {
					sourceActive = currentSource
					source(sourceActive)
				}
				current, err = fallback(0, 0)
				if err != nil {
					return err
				}
				currentSource = false
				continue
			}
			failures++
			successes = 0
			if streaming && failures >= 3 {
				streaming = false
				demand(sources.Demand{})
				if sourceActive {
					sourceActive = false
					source(false)
				}
			}
		}
	}
}

func (o *pcmOutput) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.handle != nil {
		C.nk_pcm_close(o.handle)
		o.handle = nil
	}
	return nil
}

func errno() error {
	return fmt.Errorf("errno %d", int(C.nk_errno()))
}

func syscallError(code C.int) error {
	return fmt.Errorf("errno %d", -int(code))
}
