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
#include <sound/asound.h>
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
	// The frame index S_FMT was last called with, and whether a commit has
	// since asked for a different one. The host is free to re-commit while the
	// stream runs, and without these the gadget keeps the old geometry while
	// the host decodes against the new one.
	unsigned int formatted;
	int refmt;
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
	int rc = nk_ioctl(u->fd, VIDIOC_S_FMT, &format);
	if (rc == 0) u->formatted = frame;
	return rc;
}

static void nk_uvc_close(struct nk_uvc *u);
static int nk_uvc_queue(struct nk_uvc *u, const void *data, size_t length, int all);

// The streaming layer never opens the node: it dups the descriptor the manager
// holds for the lifetime of the function (see hold.go). dup shares the struct
// file, so uvc_v4l2_open() and uvc_v4l2_release() - the two halves of
// cdev->deactivations, which do not cancel out when they overlap - run exactly
// once per linked function no matter how often streaming starts and stops.
static struct nk_uvc *nk_uvc_adopt(int held, const uint16_t *widths,
	const uint16_t *heights, const uint32_t *intervals, unsigned int frame_count,
	uint32_t max_packet) {
	if (frame_count == 0 || frame_count > NK_FRAMES || held < 0) { errno = EINVAL; return NULL; }
	struct nk_uvc *u = calloc(1, sizeof(*u));
	if (!u) return NULL;
	u->fd = fcntl(held, F_DUPFD_CLOEXEC, 0);
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
	// A commit while the stream is down is applied here. A commit while it is up
	// cannot be: S_FMT is rejected on a streaming node, and the buffers are
	// sized for the old geometry anyway. Record that the format is stale and let
	// nk_uvc_step tear the stream down and hand the caller the same edge a
	// STREAMON gives, which is already wired to re-encode at the committed size,
	// restart, and tell the source what to produce.
	if (u->pending == UVC_VS_COMMIT_CONTROL) {
		if (!u->streaming) {
			if (nk_uvc_format(u, frame) < 0) return -errno;
		} else if (frame != u->formatted) {
			u->refmt = 1;
		}
	}
	u->pending = 0;
	return 0;
}

static int nk_uvc_step(struct nk_uvc *u, const void *data, size_t length,
	int timeout_ms, int submit, unsigned int *width, unsigned int *height, unsigned int *fps) {
	struct pollfd descriptor = { .fd = u->fd, .events = POLLPRI | (u->streaming && submit ? POLLOUT : 0) };
	int polled = poll(&descriptor, 1, timeout_ms);
	if (polled < 0) return errno == EINTR ? 0 : -errno;
	int edge = 0;
	if (descriptor.revents & POLLPRI) {
		for (unsigned int handled = 0; handled < 64; handled++) {
			struct v4l2_event event;
			memset(&event, 0, sizeof(event));
			if (nk_ioctl(u->fd, VIDIOC_DQEVENT, &event) < 0) {
				// An empty event queue is ENOENT, not EAGAIN: v4l2_event_dequeue
				// returns -ENOENT when fh->available is empty, and the kernel's own
				// blocking wrapper loops on exactly that. Treating it as fatal ends
				// the worker on its first idle poll with "write UVC: errno 2", which
				// is a camera that enumerates on the host and never sends a frame.
				if (errno == EAGAIN || errno == ENOENT) break;
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
	// Taking the stream down here rather than at the commit keeps every ioctl on
	// this one thread, and leaves the restart to the caller, which is the only
	// side that can produce a frame at the newly committed size.
	if (u->refmt) {
		u->refmt = 0;
		if (u->streaming) {
			enum v4l2_buf_type stale = V4L2_BUF_TYPE_VIDEO_OUTPUT;
			nk_ioctl(u->fd, VIDIOC_STREAMOFF, &stale);
			u->streaming = 0;
			u->queued = 0;
		}
		edge = 3;
	}
	if (descriptor.revents & (POLLHUP | POLLNVAL)) return -EIO;
	if (u->streaming && (descriptor.revents & POLLERR)) return -EIO;
	unsigned int frame = u->commit.bFrameIndex > 0 ? u->commit.bFrameIndex - 1 : 0;
	if (frame >= u->frame_count) frame = 0;
	*width = u->frames[frame].width;
	*height = u->frames[frame].height;
	*fps = u->commit.dwFrameInterval ? 10000000 / u->commit.dwFrameInterval : 0;
	if (!u->streaming || !submit || !(descriptor.revents & POLLOUT)) return edge;
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

// Closes the dup, never the held descriptor: the V4L2 handle stays open, so the
// gadget stays on the bus while streaming is down.
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
// tinyalsa's own flags. PCM_IN opens /dev/snd/pcmC%uD%uc, the capture substream
// u_audio.c:645 creates for c_chmask - what the host wrote to the USB OUT
// endpoint. Never confuse it with the microphone's PCM_OUT playback substream.
enum nk_pcm_flags { NK_PCM_OUT = 0, NK_PCM_IN = 0x10000000 };
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
	int (*read)(void *, void *, unsigned int);
	int (*wait)(void *, int);
	int (*prepare)(void *);
	int (*start)(void *);
	int (*stop)(void *);
	int (*close)(void *);
	int fd;
	int capture;
};

static struct nk_pcm *nk_pcm_open(unsigned int card, unsigned int device, int capture) {
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
	int (*read_pcm)(void *, void *, unsigned int) = dlsym(library, "pcm_read");
	int (*wait_pcm)(void *, int) = dlsym(library, "pcm_wait");
	int (*prepare_pcm)(void *) = dlsym(library, "pcm_prepare");
	int (*start_pcm)(void *) = dlsym(library, "pcm_start");
	int (*stop_pcm)(void *) = dlsym(library, "pcm_stop");
	int (*close_pcm)(void *) = dlsym(library, "pcm_close");
	int (*poll_fd)(void *) = dlsym(library, "pcm_get_poll_fd");
	if (!open_pcm || !is_ready || !write_pcm || !wait_pcm || !prepare_pcm || !stop_pcm || !close_pcm) { dlclose(library); errno = ENOSYS; return NULL; }
	if (capture && (!read_pcm || !start_pcm || !poll_fd)) { dlclose(library); errno = ENOSYS; return NULL; }
	// A capture ring overruns when the reader falls behind, and the reader here
	// is a 20 ms Go ticker: four periods is 80 ms of slack, which ordinary
	// scheduling jitter eats. Eight buys 160 ms and costs 3 KiB.
	struct nk_pcm_config config = { .channels = 1, .rate = 48000, .period_size = 960,
		.period_count = capture ? 8 : 4, .format = NK_PCM_S16_LE, .avail_min = 960 };
	void *handle = open_pcm(card, device, capture ? NK_PCM_IN : NK_PCM_OUT, &config);
	if (!handle || !is_ready(handle)) { if (handle) close_pcm(handle); dlclose(library); errno = ENODEV; return NULL; }
	struct nk_pcm *pcm = calloc(1, sizeof(*pcm));
	if (!pcm) { close_pcm(handle); dlclose(library); return NULL; }
	pcm->library = library; pcm->handle = handle; pcm->write = write_pcm; pcm->read = read_pcm;
	pcm->wait = wait_pcm; pcm->prepare = prepare_pcm; pcm->start = start_pcm;
	pcm->stop = stop_pcm; pcm->close = close_pcm; pcm->fd = -1; pcm->capture = capture;
	if (capture) {
		// The descriptor is polled directly rather than through pcm_wait: this
		// tinyalsa polls POLLOUT, which a capture stream never raises. It is
		// also forced non-blocking, so that even a poll that lies cannot leave
		// pcm_read waiting inside the gadget's only reader.
		pcm->fd = poll_fd(handle);
		if (pcm->fd < 0) { close_pcm(handle); dlclose(library); free(pcm); errno = ENOTSUP; return NULL; }
		int flags = fcntl(pcm->fd, F_GETFL, 0);
		if (flags >= 0) fcntl(pcm->fd, F_SETFL, flags | O_NONBLOCK);
		if (start_pcm(handle) < 0) { prepare_pcm(handle); start_pcm(handle); }
	}
	return pcm;
}

static int nk_pcm_write(struct nk_pcm *pcm, const void *data, unsigned int length) {
	int ready = pcm->wait(pcm->handle, 0);
	if (ready == 0) return -EAGAIN;
	if (ready < 0) return ready;
	int rc = pcm->write(pcm->handle, data, length);
	return rc < 0 ? rc : 0;
}

// Read what is there and say how much that was, in frames. The transfer ioctl
// is issued directly rather than through pcm_read because pcm_read cannot
// answer the only question that matters: this tinyalsa's pcm_open writes
// avail_min = 1 into the kernel's sw_params whatever the config asks for
// (/proc/asound/.../sw_params reads back 1), so poll(POLLIN) fires on a single
// frame, the descriptor is non-blocking, and a read of a whole period comes
// back short. pcm_read returns 0 for that, and the caller then sends a period
// whose tail is still the previous one - a seam at a whole millisecond inside
// every packet, which is the buzz the target host's audio arrived with. The
// ioctl reports the frames it moved, so a short read is simply the start of a
// period the next call finishes.
static int nk_pcm_read_frames(struct nk_pcm *pcm, void *data, unsigned int frames) {
	struct snd_xferi xfer;
	xfer.result = 0;
	xfer.buf = data;
	xfer.frames = frames;
	int rc;
	do { rc = ioctl(pcm->fd, SNDRV_PCM_IOCTL_READI_FRAMES, &xfer); } while (rc < 0 && errno == EINTR);
	if (rc < 0) return -errno;
	if (xfer.result <= 0) return -EAGAIN;
	return (int)xfer.result;
}

// Stop, prepare, and for a capture stream start again, which is what puts
// tinyalsa's own prepared/running flags back in step with the kernel after an
// overrun. Nothing here drains: a stop path must never wait on a PCM.
static int nk_pcm_reset(struct nk_pcm *pcm) {
	int rc = pcm->stop(pcm->handle);
	if (rc < 0) return rc;
	rc = pcm->prepare(pcm->handle);
	if (rc < 0 || !pcm->capture) return rc;
	return pcm->start(pcm->handle);
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
	if spec.Kind == sources.KindSpeaker {
		return nil, fmt.Errorf("%s is a speaker, which is opened for capture", spec.ID)
	}
	return openPCM(node, false)
}

func (platformFactory) OpenInput(spec SlotSpec, node string) (Input, error) {
	if spec.Kind != sources.KindSpeaker {
		return nil, fmt.Errorf("%s is a %s, which is opened for playback", spec.ID, spec.Kind)
	}
	output, err := openPCM(node, true)
	if err != nil {
		return nil, err
	}
	return &pcmCapture{pcm: output}, nil
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
	if spec.FD <= 0 {
		return nil, fmt.Errorf("open UVC %s: no held descriptor", node)
	}
	handle := C.nk_uvc_adopt(C.int(spec.FD), &widths[0], &heights[0], &intervals[0], C.uint(len(frames)), C.uint32_t(spec.Video.StreamingMaxPacket))
	if handle == nil {
		return nil, fmt.Errorf("adopt UVC %s: %w", node, errno())
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
	var pace pacer
	var sourceAt time.Time
	sourceActive := false
	for {
		select {
		case <-ctx.Done():
			return nil
		case frame := <-frames:
			changed := frame.Generation != generation
			if frame.Reset || len(frame.Data) == 0 {
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
				base, size := packetSpan(current)
				rc := C.nk_uvc_restart(o.handle, unsafe.Pointer(base), C.size_t(size))
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
		timeout, submit := pace.due(time.Now(), int(fps))
		send := C.int(0)
		if submit {
			send = 1
		}
		o.mu.Lock()
		if o.handle == nil {
			o.mu.Unlock()
			return nil
		}
		base, size := packetSpan(current)
		rc := C.nk_uvc_step(o.handle, unsafe.Pointer(base), C.size_t(size), C.int(timeout), send, &width, &height, &fps)
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
			base, size = packetSpan(current)
			rc = C.nk_uvc_start(o.handle, unsafe.Pointer(base), C.size_t(size))
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

func openPCM(node string, capture bool) (*pcmOutput, error) {
	card, device, err := parseALSANode(node)
	if err != nil {
		return nil, err
	}
	direction := C.int(0)
	if capture {
		direction = 1
	}
	handle := C.nk_pcm_open(C.uint(card), C.uint(device), direction)
	if handle == nil {
		return nil, fmt.Errorf("open PCM %s: %w", node, errno())
	}
	return &pcmOutput{handle: handle}, nil
}

// pcmCapture reads the gadget's UAC2 capture substream - what the target host
// played into the USB OUT endpoint - and hands each 20 ms period to the
// manager. It polls on the same 20 ms cadence the microphone writes on, and
// every step of it is non-blocking: a host that is not playing costs an empty
// tick, never a wait.
type pcmCapture struct {
	pcm *pcmOutput
}

func (c *pcmCapture) Run(ctx context.Context, emit func(Packet), demand func(sources.Demand), active func(bool)) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	buffer := make([]byte, pcmPacketBytes)
	// Bytes of buffer already holding fresh samples. A period that arrives in
	// two reads is one period, not two, and the remainder must not be sent as
	// if it were audio.
	filled := 0
	streaming, sourceActive := false, false
	successes, failures, resets := 0, 0, 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Take every period the device has ready, not one per tick. The
			// gadget's audio clock and this ticker are independent, so a tick
			// that lands late leaves a period queued and the ring eventually
			// overruns. Bounded, so a device handing back frames forever
			// cannot hold the loop. Draining is only safe now that a short
			// read is recognised as one: doing it against a reader that
			// assumed every read was a whole period is what silenced the
			// speaker the first time it was tried.
			var rc C.int
			moved, emitted := false, 0
			for emitted < maxCaptureDrain {
				c.pcm.mu.Lock()
				if c.pcm.handle == nil {
					c.pcm.mu.Unlock()
					return nil
				}
				rc = C.nk_pcm_read_frames(c.pcm.handle, unsafe.Pointer(&buffer[filled]), C.uint((len(buffer)-filled)/2))
				c.pcm.mu.Unlock()
				if rc <= 0 {
					break
				}
				moved = true
				filled += int(rc) * 2
				if filled < len(buffer) {
					continue
				}
				emit(Packet{Data: append([]byte(nil), buffer...)})
				filled, emitted = 0, emitted+1
			}
			if moved {
				failures, resets = 0, 0
				successes++
				if !streaming && successes > 4 {
					streaming = true
					demand(sources.Demand{Streaming: true, Since: time.Now().UTC()})
				}
				if !sourceActive {
					sourceActive = true
					active(true)
				}
				continue
			}
			switch classifyPCM(int(rc)) {
			case pcmIdle:
				// Nothing to read this tick: the ordinary quiet host.
				successes = 0
				if streaming {
					failures++
				}
				if streaming && failures >= 25 {
					streaming, failures = false, 0
					demand(sources.Demand{})
					// Half a second of nothing is a host that stopped.
					// Whatever part-period is in hand belongs to the
					// stream that ended and must not be glued to the
					// front of the next one.
					filled = 0
				}
				if sourceActive && failures >= 5 {
					sourceActive = false
					active(false)
				}
				continue
			}
			successes = 0
			// The ring lost its place, so the part-period in hand no longer
			// joins what comes next.
			filled = 0
			// Anything else is an overrun or a stream the host tore down.
			// Recovery is bounded: a stream that will not come back is an
			// error the caller reports rather than a loop that spins forever.
			resets++
			if resets > 50 {
				return fmt.Errorf("read PCM: return %d after %d resets", int(rc), resets)
			}
			c.pcm.mu.Lock()
			if c.pcm.handle == nil {
				c.pcm.mu.Unlock()
				return nil
			}
			reset := C.nk_pcm_reset(c.pcm.handle)
			c.pcm.mu.Unlock()
			if reset < 0 {
				return fmt.Errorf("reset capture PCM: return %d", int(reset))
			}
			if streaming {
				streaming = false
				demand(sources.Demand{})
			}
			if sourceActive {
				sourceActive = false
				active(false)
			}
		}
	}
}

func (c *pcmCapture) Close() error { return c.pcm.Close() }

func (o *pcmOutput) Run(ctx context.Context, frames <-chan Packet, fallback Fallback, demand func(sources.Demand), source func(bool)) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	current, err := fallback(0, 0)
	if err != nil {
		return err
	}
	streaming, failures, successes, resets := false, 0, 0, 0
	sourceActive := false
	currentSource := false
	var generation uint64
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Only take a new packet once the one in hand has gone out. A tick
			// whose write found no room used to take the next packet anyway
			// and overwrite the packet it was still holding, so every full
			// ring cost the target host 20 ms of audio that the browser had
			// already been told was accepted.
			if !currentSource {
				select {
				case frame := <-frames:
					changed := frame.Generation != generation
					if frame.Reset || len(frame.Data) == 0 {
						current, err = fallback(0, 0)
						if err != nil {
							return err
						}
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
			}
			o.mu.Lock()
			if o.handle == nil {
				o.mu.Unlock()
				return nil
			}
			base, size := packetSpan(current)
			rc := C.nk_pcm_write(o.handle, unsafe.Pointer(base), C.uint(size))
			o.mu.Unlock()
			switch classifyPCM(int(rc)) {
			case pcmTransferred:
				failures, resets = 0, 0
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
			case pcmIdle:
				// The ring is full: the target host is not draining the
				// endpoint. The packet stays in current and goes out on a
				// later tick, so a host that pauses costs no audio.
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
				continue
			}
			// An underrun. This loop is paced by a Go ticker and the endpoint
			// by the host's clock, so a tick that lands late empties a ring
			// only four periods deep and ALSA parks the substream in XRUN.
			// Nothing put it back, so the first late tick silenced the
			// microphone for the rest of the binding while every frame was
			// still acknowledged - the queue filled, the writer never drained
			// it, and hw_ptr never moved again.
			successes = 0
			resets++
			if resets > 50 {
				return fmt.Errorf("write PCM: return %d after %d resets", int(rc), resets)
			}
			o.mu.Lock()
			if o.handle == nil {
				o.mu.Unlock()
				return nil
			}
			reset := C.nk_pcm_reset(o.handle)
			o.mu.Unlock()
			if reset < 0 {
				return fmt.Errorf("reset playback PCM: return %d", int(reset))
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
