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
#include <time.h>
#include <unistd.h>

#define NK_BUFFERS 4
#define NK_FRAMES 8
#define NK_MARKS 32

// The edges of a stream as this side sees them, stamped on both clocks: the
// kernel's (the event's own timestamp, or the buffer's completion time, both
// CLOCK_MONOTONIC) and this side's at the moment it was done handling. Go
// drains them after every step and writes one kernel-log line per mark, so a
// device log shows where the time between the host's alt 1 and the first
// frame went. Nothing here marks a frame.
enum nk_mark_kind { NK_MARK_EVENT = 1, NK_MARK_FIRST_SENT = 2, NK_MARK_GROUNDED = 3 };
enum nk_ground_reason { NK_GROUND_POLLERR = 1, NK_GROUND_ENODEV = 2, NK_GROUND_REFMT = 3 };
struct nk_mark {
	uint32_t kind;
	uint32_t type;      // v4l2 event type, or the ground reason
	uint32_t request;   // bRequest of a SETUP
	uint32_t selector;  // wValue >> 8 of a SETUP, or the control a DATA landed in
	uint32_t length;    // wLength, the DATA length, or the bytes of the first frame sent
	uint32_t frame;     // the frame index the control holds after a DATA
	uint32_t interval;  // the interval the control holds after a DATA
	int64_t kernel_ns;
	int64_t handled_ns;
};

static int64_t nk_mono_ns(void) {
	struct timespec now;
	clock_gettime(CLOCK_MONOTONIC, &now);
	return (int64_t)now.tv_sec * 1000000000LL + now.tv_nsec;
}

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
	struct nk_mark marks[NK_MARKS];
	unsigned int mark_count;
	unsigned int marks_lost;
	// Set by a start; the first buffer the kernel hands back after it is the
	// first frame that went out on the wire, and is marked once.
	int first_sent_pending;
	// nk_uvc_start's clock: entry, after S_FMT, after REQBUFS, after the
	// mmaps, after the QBUFs, after STREAMON.
	int64_t start_ns[6];
};

static struct nk_mark *nk_mark(struct nk_uvc *u, uint32_t kind, uint32_t type, int64_t kernel_ns) {
	if (u->mark_count >= NK_MARKS) { u->marks_lost++; return NULL; }
	struct nk_mark *m = &u->marks[u->mark_count++];
	memset(m, 0, sizeof(*m));
	m->kind = kind;
	m->type = type;
	m->kernel_ns = kernel_ns;
	m->handled_ns = nk_mono_ns();
	return m;
}

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

// The host's SET_INTERFACE to alt 1 is answered with a delayed status: f_uvc
// holds the control transfer's status stage until the STREAMON ioctl below,
// so everything between the event and that ioctl is time the host spends
// waiting on this side. start_ns records each step so the kernel log can say
// which one.
static int nk_uvc_start(struct nk_uvc *u, const void *data, size_t length) {
	unsigned int frame = u->commit.bFrameIndex > 0 ? u->commit.bFrameIndex - 1 : 0;
	if (frame >= u->frame_count) frame = 0;
	memset(u->start_ns, 0, sizeof(u->start_ns));
	u->start_ns[0] = nk_mono_ns();
	nk_uvc_release_buffers(u);
	if (nk_uvc_format(u, frame) < 0) return -errno;
	u->start_ns[1] = nk_mono_ns();
	struct v4l2_requestbuffers request;
	memset(&request, 0, sizeof(request));
	request.count = NK_BUFFERS;
	request.type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
	request.memory = V4L2_MEMORY_MMAP;
	if (nk_ioctl(u->fd, VIDIOC_REQBUFS, &request) < 0) return -errno;
	if (request.count == 0) return -ENOBUFS;
	u->start_ns[2] = nk_mono_ns();
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
	u->start_ns[3] = nk_mono_ns();
	int rc = nk_uvc_queue(u, data, length, 1);
	if (rc < 0) return rc;
	u->start_ns[4] = nk_mono_ns();
	enum v4l2_buf_type type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
	if (nk_ioctl(u->fd, VIDIOC_STREAMON, &type) < 0) return -errno;
	u->start_ns[5] = nk_mono_ns();
	u->streaming = 1;
	u->first_sent_pending = 1;
	return 0;
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

// Puts the stream back on the ground and asks the caller to raise it again.
//
// The gadget marks its buffer queue disconnected from the completion handler
// the moment a request retires with -ESHUTDOWN, which is every request in
// flight when the host resets the bus or the composite layer disables the
// function. From then on uvcg_queue_buffer refuses every QBUF with -ENODEV,
// and the flag is cleared in exactly one place - uvcg_queue_enable(queue, 0),
// which only VIDIOC_STREAMOFF reaches. Handing that ENODEV back to Go ended
// the worker, and nothing restarts a worker: the node stayed held, so the
// camera kept enumerating on the host and never sent another frame for the
// rest of the boot. Streaming off here clears the flag, and edge 3 is the same
// edge a STREAMON gives, which the caller already knows how to answer.
static int nk_uvc_ground(struct nk_uvc *u, uint32_t reason) {
	int64_t at = nk_mono_ns();
	if (u->streaming) {
		enum v4l2_buf_type type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
		nk_ioctl(u->fd, VIDIOC_STREAMOFF, &type);
	}
	u->streaming = 0;
	u->queued = 0;
	nk_mark(u, NK_MARK_GROUNDED, reason, at);
	return 3;
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
			int64_t queued_ns = (int64_t)event.timestamp.tv_sec * 1000000000LL + event.timestamp.tv_nsec;
			uint32_t pending = u->pending;
			// A control transfer this side cannot finish is that one transfer
			// lost, and the host retries it or moves on. The response ioctl
			// fails once the host has abandoned the request - a bus reset or a
			// suspend landing in the middle of a probe - and a format the
			// commit cannot apply is applied again at the STREAMON that
			// follows. Neither is a reason to end the worker, which took the
			// stream down while the host still had the camera open.
			switch (event.type) {
			case UVC_EVENT_SETUP: nk_uvc_setup(u, &uvc->req); break;
			case UVC_EVENT_DATA: nk_uvc_data(u, &uvc->data); break;
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
			// Marked after the handling, so a STREAMOFF's mark says when the
			// stream was down and not only when the event arrived.
			struct nk_mark *m = nk_mark(u, NK_MARK_EVENT, event.type, queued_ns);
			if (m && event.type == UVC_EVENT_SETUP) {
				m->request = uvc->req.bRequest;
				m->selector = uvc->req.wValue >> 8;
				m->length = uvc->req.wLength;
			} else if (m && event.type == UVC_EVENT_DATA) {
				struct uvc_streaming_control *control = pending == UVC_VS_COMMIT_CONTROL ? &u->commit : &u->probe;
				m->selector = pending;
				m->length = uvc->data.length > 0 ? (uint32_t)uvc->data.length : 0;
				m->frame = control->bFrameIndex;
				m->interval = control->dwFrameInterval;
			}
		}
	}
	// Taking the stream down here rather than at the commit keeps every ioctl on
	// this one thread, and leaves the restart to the caller, which is the only
	// side that can produce a frame at the newly committed size.
	if (u->refmt) {
		u->refmt = 0;
		if (u->streaming) nk_uvc_ground(u, NK_GROUND_REFMT);
		edge = 3;
	}
	if (descriptor.revents & (POLLHUP | POLLNVAL)) return -EIO;
	// vb2 answers a poll on a queue that is not streaming, or one it has marked
	// errored, with POLLERR alone. Both mean this side and the kernel disagree
	// about whether the stream is up, and the cure for that is to take it down
	// and raise it again rather than to end the worker.
	if (u->streaming && (descriptor.revents & POLLERR)) return nk_uvc_ground(u, NK_GROUND_POLLERR);
	unsigned int frame = u->commit.bFrameIndex > 0 ? u->commit.bFrameIndex - 1 : 0;
	if (frame >= u->frame_count) frame = 0;
	*width = u->frames[frame].width;
	*height = u->frames[frame].height;
	*fps = u->commit.dwFrameInterval ? 10000000 / u->commit.dwFrameInterval : 0;
	if (!u->streaming || !submit || !(descriptor.revents & POLLOUT)) return edge;
	// One REQBUFS sizes every buffer alike, so buffer zero answers for all of
	// them and the frame can be measured before a buffer is committed to it. A
	// payload too big for this geometry is one the source encoded against the
	// previous commit, which is exactly what arrives in the moments after the
	// host changes resolution. It costs a frame to drop; it used to cost the
	// camera, because -EMSGSIZE ended the worker and nothing restarts one.
	if (u->count > 0 && length > u->lengths[0]) return edge;
	struct v4l2_buffer buffer;
	memset(&buffer, 0, sizeof(buffer));
	buffer.type = V4L2_BUF_TYPE_VIDEO_OUTPUT;
	buffer.memory = V4L2_MEMORY_MMAP;
	if (nk_ioctl(u->fd, VIDIOC_DQBUF, &buffer) < 0) {
		if (errno == EAGAIN) return edge;
		if (errno == ENODEV) return nk_uvc_ground(u, NK_GROUND_ENODEV);
		return -errno;
	}
	if (buffer.index >= u->count) return -EIO;
	// The first buffer back after a start is the first frame the kernel has
	// finished sending; its timestamp is the completion time.
	if (u->first_sent_pending) {
		u->first_sent_pending = 0;
		int64_t completed_ns = (int64_t)buffer.timestamp.tv_sec * 1000000000LL + (int64_t)buffer.timestamp.tv_usec * 1000LL;
		struct nk_mark *m = nk_mark(u, NK_MARK_FIRST_SENT, 0, completed_ns);
		if (m) m->length = buffer.bytesused;
	}
	// The buffer is dequeued now, so it has to go back either way: an oversized
	// frame is returned empty rather than kept, which the host reads as one
	// short frame instead of a stalled endpoint.
	buffer.bytesused = 0;
	if (length <= u->lengths[buffer.index]) {
		memcpy(u->maps[buffer.index], data, length);
		buffer.bytesused = length;
	}
	if (nk_ioctl(u->fd, VIDIOC_QBUF, &buffer) < 0) {
		if (errno == ENODEV) return nk_uvc_ground(u, NK_GROUND_ENODEV);
		return -errno;
	}
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
	unsigned int frame_bytes;
	unsigned int period_bytes;
	void *silence;
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
	// Both rings are served by the same 20 ms Go ticker and drained or filled by
	// the target host's own clock, so both need the same slack: four periods is
	// 80 ms, which ordinary scheduling jitter on this board eats. Eight buys
	// 160 ms and costs 3 KiB. The playback side kept four for a while and paid
	// for it - the microphone underran about three times a second, 100 ring
	// resets in a 35 second recording, each one costing the target host 20 to
	// 40 ms of silence.
	struct nk_pcm_config config = { .channels = 1, .rate = 48000, .period_size = 960,
		.period_count = 8, .format = NK_PCM_S16_LE, .avail_min = 960 };
	void *handle = open_pcm(card, device, capture ? NK_PCM_IN : NK_PCM_OUT, &config);
	if (!handle || !is_ready(handle)) { if (handle) close_pcm(handle); dlclose(library); errno = ENODEV; return NULL; }
	struct nk_pcm *pcm = calloc(1, sizeof(*pcm));
	if (!pcm) { close_pcm(handle); dlclose(library); return NULL; }
	pcm->library = library; pcm->handle = handle; pcm->write = write_pcm; pcm->read = read_pcm;
	pcm->wait = wait_pcm; pcm->prepare = prepare_pcm; pcm->start = start_pcm;
	pcm->stop = stop_pcm; pcm->close = close_pcm; pcm->fd = -1; pcm->capture = capture;
	pcm->frame_bytes = config.channels * 2;
	pcm->period_bytes = config.period_size * pcm->frame_bytes;
	// One period of zeros, kept for priming the ring after a reset. A failed
	// allocation only costs the cushion, so it is not worth failing the open.
	if (!capture) pcm->silence = calloc(1, pcm->period_bytes);
	// Playback needs the descriptor for two things: asking the kernel how much
	// of the ring is free, and refusing to sleep in it. Checking the room first
	// is not enough on its own, because the target host can stop draining the
	// isochronous endpoint between the check and the write - and then a
	// blocking write parks the microphone's only writer inside
	// snd_pcm_lib_write for ALSA's ten second timeout, with the loop unable to
	// tick, reset, or publish that the stream has stopped. Measured on the
	// device: hw_ptr frozen for 41 seconds, the sink still reporting demand
	// true and output silence, and the browser's frames still being accepted
	// and acknowledged into a queue nobody was draining. O_NONBLOCK turns that
	// into an EAGAIN the loop already knows how to sit out. The write cannot
	// come back partial: nk_pcm_room has already seen a whole period free, and
	// a playback ring's free space only ever grows until this loop writes.
	if (!capture && poll_fd) {
		pcm->fd = poll_fd(handle);
		if (pcm->fd >= 0) {
			int flags = fcntl(pcm->fd, F_GETFL, 0);
			if (flags >= 0) fcntl(pcm->fd, F_SETFL, flags | O_NONBLOCK);
		}
	}
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

// nk_pcm_room answers, from the kernel rather than from poll(), whether a whole
// `frames`-frame write can go out right now: NK_PCM_ROOM_YES, NK_PCM_ROOM_NO,
// NK_PCM_ROOM_UNKNOWN when the ioctl cannot be asked, or a negative errno for a
// stream that has to be reset before it will move another sample.
//
// pcm_wait cannot answer either half of that. pcm_open leaves the kernel's
// avail_min at one frame whatever the config asked for - the same quirk
// nk_pcm_read_frames documents from the capture side, and
// /proc/asound/.../sw_params reads back avail_min: 1 - so POLLOUT means only
// that one frame of a 3840 frame ring is free, not that a 960 frame period
// fits. And a parked substream raises no POLLOUT at all, which pcm_wait
// reports as a plain timeout, indistinguishable from a ring that is merely
// full. Only the state field tells those two apart.
#define NK_PCM_ROOM_NO 0
#define NK_PCM_ROOM_YES 1
#define NK_PCM_ROOM_UNKNOWN 2
static int nk_pcm_room(struct nk_pcm *pcm, unsigned int frames) {
	struct snd_pcm_status status;
	memset(&status, 0, sizeof(status));
	if (pcm->fd < 0 || ioctl(pcm->fd, SNDRV_PCM_IOCTL_STATUS, &status) < 0) return NK_PCM_ROOM_UNKNOWN;
	switch (status.state) {
	case SNDRV_PCM_STATE_XRUN: return -EPIPE;
	case SNDRV_PCM_STATE_SUSPENDED: return -ESTRPIPE;
	case SNDRV_PCM_STATE_DISCONNECTED: return -ENODEV;
	default: break;
	}
	return status.avail >= (snd_pcm_sframes_t)frames ? NK_PCM_ROOM_YES : NK_PCM_ROOM_NO;
}

// nk_pcm_probe reports what the kernel says about the stream, for the log line
// the playback loop prints when it has been unable to write for a while: the
// state in the low byte, avail in the rest, or a negative errno if the ioctl
// this all rests on is not answered at all.
static long nk_pcm_probe(struct nk_pcm *pcm) {
	struct snd_pcm_status status;
	memset(&status, 0, sizeof(status));
	if (pcm->fd < 0) return -ENOTTY;
	if (ioctl(pcm->fd, SNDRV_PCM_IOCTL_STATUS, &status) < 0) return -errno;
	return ((long)status.avail << 8) | (status.state & 0xff);
}

// The kernel is asked first, and pcm_wait is only the fallback for a kernel
// that will not answer, because the two failures pcm_wait hides are the two
// that matter here.
//
// A parked substream raises no POLLOUT, so pcm_wait returns a timeout and the
// old code turned that into -EAGAIN. -EAGAIN is classifyPCM's pcmIdle, and the
// idle branch of the playback loop deliberately does nothing but wait for the
// next tick - it never resets. So the first underrun parked the ring in XRUN
// for good: the loop ticked on at 50 Hz writing nothing, hw_ptr never moved
// again, and the microphone was silent for the rest of the session while the
// browser's frames were still being accepted and acknowledged into a queue
// nobody drained. Measured on the device, /proc/asound/card2/pcm0p/sub0/status
// sat in XRUN with hw_ptr == appl_ptr and trigger_time 541 seconds stale while
// a browser pushed 50 packets a second at it, and the sink reported exactly
// the state the bug was reported as: demand true, output silence, binding
// claimed. Reporting the parked stream as -EPIPE instead sends the loop down
// the reset path it already has.
//
// The other failure is the opposite one. The playback descriptor is blocking,
// and pcm_open leaves avail_min at one frame, so POLLOUT is raised when a
// single frame of a 3840 frame ring is free while the write that follows asks
// for a whole 960 frame period. That write sleeps inside snd_pcm_lib_write
// until the host drains enough, and a host that has stopped draining never
// does - the microphone's only writer then lives in that syscall until ALSA's
// own ten second timeout. Refusing the tick unless a whole period already fits
// leaves the blocking write nothing to wait for.
static int nk_pcm_write(struct nk_pcm *pcm, const void *data, unsigned int length) {
	unsigned int frames = pcm->frame_bytes ? length / pcm->frame_bytes : length;
	int room = nk_pcm_room(pcm, frames);
	if (room < 0) return room;
	if (room == NK_PCM_ROOM_UNKNOWN) {
		int ready = pcm->wait(pcm->handle, 0);
		if (ready == 0) return -EAGAIN;
		if (ready < 0) return ready;
	} else if (room == NK_PCM_ROOM_NO) {
		return -EAGAIN;
	}
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

// How many periods of silence a prepared playback ring is primed with, and so
// how much slack the writer runs with from then on. The writer puts one period
// in per 20 ms tick and the endpoint takes one out per 20 ms, so the level
// never climbs on its own: whatever prepare() leaves in the ring is the entire
// cushion until the next reset. Priming nothing means the first tick that lands
// late finds the ring empty and ALSA parks the substream, and the reset that
// follows primes nothing again - on the device that ran at about three
// underruns a second, 45 separate dropouts of 20 to 120 ms in one 17 second
// recording of a continuous tone, 12% of the audio missing. Four periods is
// 80 ms of scheduling jitter absorbed, half the eight period ring, leaving the
// other half as room to write into.
#define NK_PCM_PRIME_PERIODS 4

// Stop, prepare, and for a capture stream start again, which is what puts
// tinyalsa's own prepared/running flags back in step with the kernel after an
// overrun. Nothing here drains: a stop path must never wait on a PCM.
static int nk_pcm_reset(struct nk_pcm *pcm) {
	int rc = pcm->stop(pcm->handle);
	if (rc < 0) return rc;
	rc = pcm->prepare(pcm->handle);
	if (rc < 0) return rc;
	if (pcm->capture) return pcm->start(pcm->handle);
	// Priming writes into a ring prepare() just emptied, so it cannot block
	// even though the descriptor is blocking. A refusal is not fatal - the
	// stream still runs, just without the cushion - so the loop is not told.
	if (pcm->silence) {
		for (int i = 0; i < NK_PCM_PRIME_PERIODS; i++) {
			if (pcm->write(pcm->handle, pcm->silence, pcm->period_bytes) < 0) break;
		}
	}
	return 0;
}

static void nk_pcm_close(struct nk_pcm *pcm) {
	if (!pcm) return;
	pcm->close(pcm->handle);
	dlclose(pcm->library);
	free(pcm->silence);
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
	"NanoKVM-Server/service/startup"
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
	trace  *uvcTrace
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
	return &uvcOutput{handle: handle, trace: newUVCTrace(spec.ID)}, nil
}

// Every edge the C side marked since the last step, as kernel-log lines.
// Called with o.mu held and o.handle live.
func (o *uvcOutput) drainMarks() {
	count := int(o.handle.mark_count)
	for i := 0; i < count && i < len(o.handle.marks); i++ {
		m := o.handle.marks[i]
		o.trace.event(uvcMark{
			kind: uint32(m.kind), eventType: uint32(m._type), request: uint32(m.request),
			selector: uint32(m.selector), length: uint32(m.length), frame: uint32(m.frame),
			interval: uint32(m.interval), kernelNS: int64(m.kernel_ns), handledNS: int64(m.handled_ns),
		})
	}
	o.handle.mark_count = 0
	if lost := int(o.handle.marks_lost); lost > 0 {
		o.handle.marks_lost = 0
		o.trace.emit("uvc %s: %d edges were not recorded, more than %d arrived in one poll", o.trace.slot, lost, C.NK_MARKS)
	}
}

// The loop itself is runVideo in manager.go, where it is pure Go and a test can
// play the host against it. This side only carries each call into the kernel.
func (o *uvcOutput) Run(ctx context.Context, frames <-chan Packet, fallback Fallback, demand func(sources.Demand), source func(bool)) error {
	return runVideo(ctx, frames, fallback, demand, source, o)
}

func (o *uvcOutput) step(current Packet, timeoutMS int, submit bool) (videoStep, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.handle == nil {
		return videoStep{}, errVideoClosed
	}
	send := C.int(0)
	if submit {
		send = 1
	}
	var width, height, fps C.uint
	base, size := packetSpan(current)
	rc := C.nk_uvc_step(o.handle, unsafe.Pointer(base), C.size_t(size), C.int(timeoutMS), send, &width, &height, &fps)
	o.drainMarks()
	if rc < 0 {
		return videoStep{}, syscallError(rc)
	}
	return videoStep{edge: videoEdge(rc), width: int(width), height: int(height), fps: int(fps)}, nil
}

func (o *uvcOutput) start(current Packet) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.handle == nil {
		return errVideoClosed
	}
	base, size := packetSpan(current)
	rc := C.nk_uvc_start(o.handle, unsafe.Pointer(base), C.size_t(size))
	var stamps uvcStart
	for i := range stamps {
		stamps[i] = int64(o.handle.start_ns[i])
	}
	if rc < 0 {
		err := syscallError(rc)
		o.trace.started(stamps, err)
		return err
	}
	o.trace.started(stamps, nil)
	// The host's STREAMON is the edge the lost INT was measured against, so
	// Go's handler is put back here as well as at worker start.
	startup.ReassertInterrupt("camera stream start")
	return nil
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

// The playback loop itself is runPlayback in pcmloop.go, driven through these
// three calls so its pacing can be tested against a fake ring on any host.
// Each answers false once Close has run under the loop, which the loop takes
// as its cue to leave quietly.
func (o *pcmOutput) write(packet Packet) (int, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.handle == nil {
		return 0, false
	}
	base, size := packetSpan(packet)
	return int(C.nk_pcm_write(o.handle, unsafe.Pointer(base), C.uint(size))), true
}

func (o *pcmOutput) reset() (int, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.handle == nil {
		return 0, false
	}
	return int(C.nk_pcm_reset(o.handle)), true
}

func (o *pcmOutput) probe() (int64, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.handle == nil {
		return 0, false
	}
	return int64(C.nk_pcm_probe(o.handle)), true
}

func (o *pcmOutput) Run(ctx context.Context, frames <-chan Packet, fallback Fallback, demand func(sources.Demand), source func(bool)) error {
	return runPlayback(ctx, o, frames, fallback, demand, source)
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
