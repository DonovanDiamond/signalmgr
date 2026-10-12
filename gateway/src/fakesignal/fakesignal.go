// Package fakesignal is an in-memory stand-in for signal-cli-rest-api 0.101,
// serving only the endpoints the gateway's signal service uses.
package fakesignal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// ValidCode is the only verification code Verify accepts.
	ValidCode = "123456"
	// RateLimitedNumber always answers register with a 429 and challenge tokens.
	RateLimitedNumber = "+15550000429"

	captchaPrefix = "signalcaptcha://"
	linkDelay     = 5 * time.Second
)

type device struct {
	CreationTimestamp int64  `json:"creation_timestamp"`
	ID                int64  `json:"id"`
	LastSeenTimestamp int64  `json:"last_seen_timestamp"`
	Name              string `json:"name"`
}

// Server is an http.Handler holding all fake state.
type Server struct {
	mux *http.ServeMux

	mu       sync.Mutex
	order    []string // registered numbers in registration order
	devices  map[string][]device
	pending  map[string]bool // register called, not yet verified
	pins     map[string]bool
	linkWait bool // a link timer is already running
}

// New returns a fake server; seed adds a few registered numbers with devices.
func New(seed bool) *Server {
	s := &Server{
		mux:     http.NewServeMux(),
		devices: map[string][]device{},
		pending: map[string]bool{},
		pins:    map[string]bool{},
	}
	if seed {
		now := time.Now()
		ms := func(ago time.Duration) int64 { return now.Add(-ago).UnixMilli() }
		day := 24 * time.Hour
		s.add("+27821234567",
			device{ID: 1, Name: "Pixel 8", CreationTimestamp: ms(400 * day), LastSeenTimestamp: ms(time.Hour)},
			device{ID: 2, Name: "Reception desktop", CreationTimestamp: ms(120 * day), LastSeenTimestamp: ms(3 * time.Hour)},
			device{ID: 3, Name: "Office iPad", CreationTimestamp: ms(30 * day), LastSeenTimestamp: ms(2 * day)})
		s.add("+27835550142",
			device{ID: 1, Name: "Galaxy A54", CreationTimestamp: ms(200 * day), LastSeenTimestamp: ms(10 * time.Minute)})
		s.add("+27725550199",
			device{ID: 1, Name: "iPhone 14", CreationTimestamp: ms(90 * day), LastSeenTimestamp: ms(day)},
			device{ID: 2, Name: "Signal Desktop", CreationTimestamp: ms(60 * day), LastSeenTimestamp: ms(5 * day)})
	}

	s.mux.HandleFunc("GET /v1/about", s.about)
	s.mux.HandleFunc("GET /v1/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	s.mux.HandleFunc("GET /v1/accounts", s.accounts)
	s.mux.HandleFunc("POST /v1/register/{number}", s.register)
	s.mux.HandleFunc("POST /v1/register/{number}/verify/{token}", s.verify)
	s.mux.HandleFunc("POST /v1/unregister/{number}", s.unregister)
	s.mux.HandleFunc("GET /v1/devices/{number}", s.listDevices)
	s.mux.HandleFunc("DELETE /v1/devices/{number}/{id}", s.deleteDevice)
	s.mux.HandleFunc("GET /v1/qrcodelink", s.qrcode)
	s.mux.HandleFunc("GET /v1/qrcodelink/raw", s.linkURI)
	s.mux.HandleFunc("POST /v1/accounts/{number}/pin", s.setPin)
	s.mux.HandleFunc("DELETE /v1/accounts/{number}/pin", s.removePin)
	s.mux.HandleFunc("POST /v1/accounts/{number}/rate-limit-challenge", s.rateLimit)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// add must be called with no concurrent access (seeding) or with mu held.
func (s *Server) add(number string, devs ...device) {
	if !slices.Contains(s.order, number) {
		s.order = append(s.order, number)
	}
	s.devices[number] = devs
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func (s *Server) about(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"build":        1,
		"capabilities": map[string][]string{"default": {"v1", "v2"}},
		"mode":         "json-rpc",
		"version":      "0.101",
		"versions":     []string{"v1", "v2"},
	})
}

func (s *Server) accounts(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, append([]string{}, s.order...))
}

type registerBody struct {
	Captcha  string `json:"captcha"`
	UseVoice bool   `json:"use_voice"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("number")
	var b registerBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !strings.HasPrefix(b.Captcha, captchaPrefix) {
		writeErr(w, http.StatusBadRequest, "Captcha required for verification, must start with "+captchaPrefix)
		return
	}
	if number == RateLimitedNumber {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error":            "Rate Limited",
			"account":          number,
			"challenge_tokens": []string{"fake-challenge-token-1"},
		})
		return
	}
	s.mu.Lock()
	s.pending[number] = true
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) verify(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("number")
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.PathValue("token") != ValidCode || !s.pending[number] {
		writeErr(w, http.StatusBadRequest, "Verify error: invalid verification code or no registration in progress")
		return
	}
	delete(s.pending, number)
	now := time.Now().UnixMilli()
	s.add(number, device{ID: 1, Name: "Primary device", CreationTimestamp: now, LastSeenTimestamp: now})
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) unregister(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("number")
	s.mu.Lock()
	defer s.mu.Unlock()
	if !slices.Contains(s.order, number) {
		writeErr(w, http.StatusBadRequest, "User is not registered.")
		return
	}
	s.order = slices.DeleteFunc(s.order, func(n string) bool { return n == number })
	delete(s.devices, number)
	delete(s.pins, number)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	devs, ok := s.devices[r.PathValue("number")]
	if !ok {
		writeErr(w, http.StatusBadRequest, "User is not registered.")
		return
	}
	writeJSON(w, http.StatusOK, devs)
}

func (s *Server) deleteDevice(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("number")
	var id int64
	if _, err := fmt.Sscanf(r.PathValue("id"), "%d", &id); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid device id")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	devs, ok := s.devices[number]
	if !ok || id == 1 || !slices.ContainsFunc(devs, func(d device) bool { return d.ID == id }) {
		writeErr(w, http.StatusBadRequest, "No linked device with that id")
		return
	}
	s.devices[number] = slices.DeleteFunc(devs, func(d device) bool { return d.ID == id })
	w.WriteHeader(http.StatusNoContent)
}

// scheduleLink simulates the user scanning the QR code: a device appears on
// the first registered account a few seconds after the link was requested.
func (s *Server) scheduleLink() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.linkWait {
		return
	}
	s.linkWait = true
	time.AfterFunc(linkDelay, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.linkWait = false
		if len(s.order) == 0 {
			return
		}
		number := s.order[0]
		var next int64 = 1
		for _, d := range s.devices[number] {
			next = max(next, d.ID+1)
		}
		now := time.Now().UnixMilli()
		s.devices[number] = append(s.devices[number],
			device{ID: next, Name: "Linked device", CreationTimestamp: now, LastSeenTimestamp: now})
	})
}

func deviceName(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Query().Get("device_name") == "" {
		writeErr(w, http.StatusBadRequest, "device_name is required")
		return false
	}
	return true
}

func (s *Server) qrcode(w http.ResponseWriter, r *http.Request) {
	if !deviceName(w, r) {
		return
	}
	s.scheduleLink()
	w.Header().Set("Content-Type", "image/png")
	w.Write(qrPNG())
}

func (s *Server) linkURI(w http.ResponseWriter, r *http.Request) {
	if !deviceName(w, r) {
		return
	}
	s.scheduleLink()
	writeJSON(w, http.StatusOK, map[string]string{
		"device_link_uri": "sgnl://linkdevice?uuid=fake-uuid&pub_key=fake-public-key",
	})
}

func (s *Server) setPin(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("number")
	var b struct {
		Pin string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Pin == "" {
		writeErr(w, http.StatusBadRequest, "pin is required")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !slices.Contains(s.order, number) {
		writeErr(w, http.StatusBadRequest, "User is not registered.")
		return
	}
	s.pins[number] = true
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) removePin(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("number")
	s.mu.Lock()
	defer s.mu.Unlock()
	if !slices.Contains(s.order, number) {
		writeErr(w, http.StatusBadRequest, "User is not registered.")
		return
	}
	delete(s.pins, number)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) rateLimit(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Captcha        string `json:"captcha"`
		ChallengeToken string `json:"challenge_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.ChallengeToken == "" {
		writeErr(w, http.StatusBadRequest, "challenge_token is required")
		return
	}
	if !strings.HasPrefix(b.Captcha, captchaPrefix) {
		writeErr(w, http.StatusBadRequest, "captcha must start with "+captchaPrefix)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// qrPNG draws a QR-looking 29x29 module pattern; it is not scannable.
func qrPNG() []byte {
	const modules, scale = 29, 8
	img := image.NewGray(image.Rect(0, 0, (modules+2)*scale, (modules+2)*scale))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	finder := func(x, y int) bool {
		for _, o := range [][2]int{{0, 0}, {modules - 7, 0}, {0, modules - 7}} {
			dx, dy := x-o[0], y-o[1]
			if dx >= 0 && dx < 7 && dy >= 0 && dy < 7 {
				edge := dx == 0 || dx == 6 || dy == 0 || dy == 6
				return edge || (dx >= 2 && dx <= 4 && dy >= 2 && dy <= 4)
			}
		}
		return (x*7+y*13+x*y)%3 == 0
	}
	for y := range modules {
		for x := range modules {
			if !finder(x, y) {
				continue
			}
			for py := range scale {
				for px := range scale {
					img.SetGray((x+1)*scale+px, (y+1)*scale+py, color.Gray{})
				}
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
