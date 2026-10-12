// Package signal is the gateway's service over the signalmgr library: it merges
// upstream accounts with our own labels and registration state, validates
// input, and writes an audit row for every mutating action.
package signal

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/DonovanDiamond/signalmgr"
	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

const (
	StateRegistered    = "registered"
	StateAwaitingCode  = "awaiting_code"
	StateNotRegistered = "not_registered"

	captchaPrefix = "signalcaptcha://"
	maxLabel      = 64
	maxDeviceName = 64
	maxToken      = 4096

	linkURIPrefix = "sgnl://linkdevice?"
	maxLinkURI    = 2048
)

var (
	ErrInvalid     = errors.New("invalid input")
	ErrUnreachable = errors.New("signal API unreachable")

	numberRe = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
	aciRe    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	codeRe   = regexp.MustCompile(`^\d{6}$`)
	pinRe    = regexp.MustCompile(`^\d{4,20}$`)
)

type Device struct {
	ID       int64      `json:"id"`
	Name     string     `json:"name"`
	Created  *time.Time `json:"created"`
	LastSeen *time.Time `json:"lastSeen"`
	Primary  bool       `json:"primary"`
}

// Number is one account. ID is what the upstream API calls it: the phone
// number, or the lowercase ACI when the account has none (Number is then "").
type Number struct {
	ID      string   `json:"id"`
	Number  string   `json:"number"`
	ACI     string   `json:"aci"`
	Label   string   `json:"label"`
	State   string   `json:"state"`
	Devices []Device `json:"devices"`
}

type GatewayInfo struct {
	Online  bool   `json:"online"`
	Version string `json:"version"`
	Mode    string `json:"mode"`
}

type Service struct {
	db     *sqlx.DB
	apiURL string
	log    zerolog.Logger
}

// New points the signalmgr library at apiURL. The library keeps that in a
// package global, so only one Service per process is meaningful.
func New(x *sqlx.DB, apiURL string, log zerolog.Logger) *Service {
	signalmgr.API_URL = apiURL
	return &Service{db: x, apiURL: apiURL, log: log}
}

func ValidateNumber(n string) error {
	if !numberRe.MatchString(n) {
		return fmt.Errorf("%w: number must be E.164 like +27821234567, got %q", ErrInvalid, n)
	}
	return nil
}

// ParseID checks an account id taken from a URL: an E.164 number or an ACI.
// An ACI is returned lowercased, as upstream normalises it; on error the input
// is returned unchanged so the audit row shows what was sent.
func ParseID(id string) (string, error) {
	if aciRe.MatchString(id) {
		return strings.ToLower(id), nil
	}
	if numberRe.MatchString(id) {
		return id, nil
	}
	return id, fmt.Errorf("%w: account id must be an E.164 number like +27821234567 or an ACI like 5f1c2d3e-4b5a-4c6d-8e7f-90a1b2c3d4e5, got %q", ErrInvalid, id)
}

// newNumber builds the entry for an account id, putting it in the number or
// the aci field according to its form.
func newNumber(id string) *Number {
	n := &Number{ID: id, Devices: []Device{}}
	if aciRe.MatchString(id) {
		n.ACI = id
	} else {
		n.Number = id
	}
	return n
}

func validateLabel(l string) error {
	if utf8.RuneCountInString(l) > maxLabel {
		return fmt.Errorf("%w: label must be at most %d characters", ErrInvalid, maxLabel)
	}
	return nil
}

func validateCaptcha(c string) error {
	if !strings.HasPrefix(c, captchaPrefix) {
		return fmt.Errorf("%w: captcha must be the token starting with %s", ErrInvalid, captchaPrefix)
	}
	return nil
}

func validatePin(p string) error {
	if !pinRe.MatchString(p) {
		return fmt.Errorf("%w: PIN must be 4 to 20 digits", ErrInvalid)
	}
	return nil
}

func validateDeviceName(n string) error {
	if c := utf8.RuneCountInString(n); c < 1 || c > maxDeviceName {
		return fmt.Errorf("%w: device name must be 1 to %d characters", ErrInvalid, maxDeviceName)
	}
	return nil
}

func validateLinkURI(u string) error {
	if !strings.HasPrefix(u, linkURIPrefix) || len(u) > maxLinkURI {
		return fmt.Errorf("%w: link URI must start with %s and be at most %d characters", ErrInvalid, linkURIPrefix, maxLinkURI)
	}
	return nil
}

// upstream classifies a library error. The library's error text embeds the
// request path, which for verify contains the code, so connection failures are
// replaced with a fixed message instead of being passed on.
func (s *Service) upstream(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *signalmgr.APIError
	if errors.As(err, &apiErr) {
		return err
	}
	s.log.Warn().Str("api_url", s.apiURL).Msg("signal API request failed")
	return fmt.Errorf("%w: no response from %s", ErrUnreachable, s.apiURL)
}

// result is what goes in the audit row; APIError.Error() is avoided because it
// includes the request path.
func result(err error) string {
	if err == nil {
		return "ok"
	}
	var apiErr *signalmgr.APIError
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("upstream status %d: %s", apiErr.Status, apiErr.Message)
	}
	return err.Error()
}

func (s *Service) audit(ctx context.Context, actor, action, number, detail string, err error) {
	e := db.AuditEntry{At: time.Now().UTC(), Actor: actor, Action: action, Number: number, Detail: detail, Result: result(err)}
	if aerr := db.InsertAudit(ctx, s.db, e); aerr != nil {
		s.log.Error().Err(aerr).Str("actor", actor).Str("action", action).Str("number", number).Msg("writing audit row")
	}
}

func (s *Service) Gateway(ctx context.Context) GatewayInfo {
	about, err := signalmgr.GetAbout()
	if err != nil {
		return GatewayInfo{}
	}
	return GatewayInfo{Online: true, Version: about.Version, Mode: about.Mode}
}

func toDevice(d signalmgr.Device) Device {
	out := Device{ID: d.ID, Name: d.Name, Primary: d.ID == 1}
	if d.CreationTimestamp != 0 {
		t := time.UnixMilli(d.CreationTimestamp).UTC()
		out.Created = &t
	}
	if d.LastSeenTimestamp != 0 {
		t := time.UnixMilli(d.LastSeenTimestamp).UTC()
		out.LastSeen = &t
	}
	return out
}

func (s *Service) Numbers(ctx context.Context) ([]Number, error) {
	rows, err := db.ListNumbers(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("listing numbers: %w", err)
	}
	accounts, err := signalmgr.GetAccounts()
	if err != nil {
		return nil, s.upstream(err)
	}

	byID := map[string]*Number{}
	for _, r := range rows {
		n := newNumber(r.Number)
		n.Label = r.Label
		n.State = StateNotRegistered
		if r.RegisterRequestedAt != nil {
			n.State = StateAwaitingCode
		}
		byID[r.Number] = n
	}
	for _, info := range accounts {
		id := strings.ToLower(info.ID())
		n, ok := byID[id]
		if !ok {
			n = newNumber(id)
			byID[id] = n
		}
		// A numbered account may also report its ACI; keep both.
		n.ACI = strings.ToLower(info.ACI)
		n.State = StateRegistered
		acc := signalmgr.Account{Number: id}
		devs, err := acc.GetDevices()
		if err != nil {
			return nil, s.upstream(err)
		}
		for _, d := range devs {
			n.Devices = append(n.Devices, toDevice(d))
		}
	}

	out := make([]Number, 0, len(byID))
	for _, n := range byID {
		out = append(out, *n)
	}
	slices.SortFunc(out, func(a, b Number) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func (s *Service) AddNumber(ctx context.Context, actor, number, label string) error {
	err := errors.Join(ValidateNumber(number), validateLabel(label))
	if err == nil {
		err = db.InsertNumber(ctx, s.db, number, label)
	}
	s.audit(ctx, actor, "add_number", number, "label="+label, err)
	return err
}

func (s *Service) SetLabel(ctx context.Context, actor, number, label string) error {
	number, err := ParseID(number)
	err = errors.Join(err, validateLabel(label))
	if err == nil {
		err = db.UpsertNumberLabel(ctx, s.db, number, label)
	}
	s.audit(ctx, actor, "set_label", number, "label="+label, err)
	return err
}

// ensureRow makes sure the number has a DB row so its state can be tracked.
func (s *Service) ensureRow(ctx context.Context, number string) error {
	_, err := db.GetNumber(ctx, s.db, number)
	if errors.Is(err, db.ErrNotFound) {
		return db.InsertNumber(ctx, s.db, number, "")
	}
	return err
}

// clearRequested forgets a pending registration; a number with no row has none.
func (s *Service) clearRequested(ctx context.Context, number string) error {
	err := db.SetRegisterRequested(ctx, s.db, number, nil)
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	return err
}

func (s *Service) Register(ctx context.Context, actor, number, captcha string, voice bool) error {
	err := errors.Join(ValidateNumber(number), validateCaptcha(captcha))
	if err == nil {
		acc := signalmgr.Account{Number: number}
		err = s.upstream(acc.PostRegister(captcha, voice))
	}
	if err == nil {
		if err = s.ensureRow(ctx, number); err == nil {
			now := time.Now().UTC()
			err = db.SetRegisterRequested(ctx, s.db, number, &now)
		}
	}
	s.audit(ctx, actor, "register", number, fmt.Sprintf("voice=%t", voice), err)
	return err
}

func (s *Service) Verify(ctx context.Context, actor, number, code, pin string) error {
	err := errors.Join(ValidateNumber(number))
	if !codeRe.MatchString(code) {
		err = errors.Join(err, fmt.Errorf("%w: code must be 6 digits", ErrInvalid))
	}
	if pin != "" {
		err = errors.Join(err, validatePin(pin))
	}
	if err == nil {
		acc := signalmgr.Account{Number: number}
		err = s.upstream(acc.PostRegisterVerify(code, pin))
	}
	if err == nil {
		err = s.clearRequested(ctx, number)
	}
	s.audit(ctx, actor, "verify", number, fmt.Sprintf("pin_supplied=%t", pin != ""), err)
	return err
}

func (s *Service) Unregister(ctx context.Context, actor, number string, deleteAccount, deleteLocalData bool) error {
	number, err := ParseID(number)
	if err == nil {
		acc := signalmgr.Account{Number: number}
		err = s.upstream(acc.PostUnregister(struct {
			DeleteAccount   bool `json:"delete_account"`
			DeleteLocalData bool `json:"delete_local_data"`
		}{deleteAccount, deleteLocalData}))
	}
	if err == nil {
		err = s.clearRequested(ctx, number)
	}
	s.audit(ctx, actor, "unregister", number,
		fmt.Sprintf("delete_account=%t delete_local_data=%t", deleteAccount, deleteLocalData), err)
	return err
}

func (s *Service) RemoveDevice(ctx context.Context, actor, number string, id int64) error {
	number, err := ParseID(number)
	if err == nil && id < 2 {
		err = fmt.Errorf("%w: device id must be 2 or higher; device 1 is the primary", ErrInvalid)
	}
	if err == nil {
		acc := signalmgr.Account{Number: number}
		err = s.upstream(acc.DeleteDevice(id))
	}
	s.audit(ctx, actor, "remove_device", number, fmt.Sprintf("device_id=%d", id), err)
	return err
}

func (s *Service) LinkDevice(ctx context.Context, actor, number, uri string) error {
	number, err := ParseID(number)
	err = errors.Join(err, validateLinkURI(uri))
	if err == nil {
		acc := signalmgr.Account{Number: number}
		err = s.upstream(acc.PostLinkDevice(struct {
			URI string `json:"uri"`
		}{uri}))
	}
	// The URI carries a one-time key, so it is not written to the audit row.
	s.audit(ctx, actor, "link_device", number, "", err)
	return err
}

func (s *Service) SetPin(ctx context.Context, actor, number, pin string) error {
	number, err := ParseID(number)
	err = errors.Join(err, validatePin(pin))
	if err == nil {
		acc := signalmgr.Account{Number: number}
		err = s.upstream(acc.PostPin(struct {
			Pin string `json:"pin"`
		}{pin}))
	}
	s.audit(ctx, actor, "set_pin", number, "", err)
	return err
}

func (s *Service) RemovePin(ctx context.Context, actor, number string) error {
	number, err := ParseID(number)
	if err == nil {
		acc := signalmgr.Account{Number: number}
		err = s.upstream(acc.DeletePin())
	}
	s.audit(ctx, actor, "remove_pin", number, "", err)
	return err
}

func (s *Service) RateLimit(ctx context.Context, actor, number, challengeToken, captcha string) error {
	number, err := ParseID(number)
	err = errors.Join(err, validateCaptcha(captcha))
	if challengeToken == "" || len(challengeToken) > maxToken {
		err = errors.Join(err, fmt.Errorf("%w: challenge token must be 1 to %d characters", ErrInvalid, maxToken))
	}
	if err == nil {
		acc := signalmgr.Account{Number: number}
		err = s.upstream(acc.PostRateLimitChallenge(struct {
			Captcha        string `json:"captcha"`
			ChallengeToken string `json:"challenge_token"`
		}{captcha, challengeToken}))
	}
	s.audit(ctx, actor, "rate_limit_challenge", number, "", err)
	return err
}

func (s *Service) LinkQR(ctx context.Context, deviceName string) ([]byte, error) {
	if err := validateDeviceName(deviceName); err != nil {
		return nil, err
	}
	png, err := signalmgr.GetLinkAccountQRCode(deviceName, signalmgr.GetLinkAccountQRCodeOptions{})
	return png, s.upstream(err)
}

func (s *Service) LinkURI(ctx context.Context, deviceName string) (string, error) {
	if err := validateDeviceName(deviceName); err != nil {
		return "", err
	}
	uri, err := signalmgr.GetLinkAccountURI(deviceName)
	return uri, s.upstream(err)
}

func (s *Service) Audit(ctx context.Context, limit int) ([]db.AuditEntry, error) {
	return db.ListAudit(ctx, s.db, limit)
}
