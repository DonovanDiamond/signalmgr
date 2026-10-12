package auth

import (
	"context"
	"fmt"
	"sort"
	"strings"

	ldapauth "github.com/DonovanDiamond/go-ad-auth/v3"
	"github.com/rs/zerolog"

	"github.com/DonovanDiamond/signalmgr/gateway/src/config"
)

// directoryUser is what a successful directory sign-in reports.
type directoryUser struct {
	Name string
	Role string
}

// directory authenticates against Active Directory. It returns
// ErrInvalidCredentials for a refused bind, ErrNoRole when the person is in
// no mapped group, and ErrDirectoryUnreachable for every infrastructure
// failure. It exists as an interface so tests can stand in for a server.
type directory interface {
	Authenticate(ctx context.Context, username, password string) (directoryUser, error)
}

type adDirectory struct {
	cfg config.ADConfig
	ad  *ldapauth.Config
	log zerolog.Logger
}

func newADDirectory(cfg config.ADConfig, log zerolog.Logger) *adDirectory {
	sec := ldapauth.SecurityNone
	switch cfg.Security {
	case "tls":
		sec = ldapauth.SecurityTLS
	case "starttls":
		sec = ldapauth.SecurityStartTLS
	}
	return &adDirectory{
		cfg: cfg,
		ad:  &ldapauth.Config{Server: cfg.Server, Port: cfg.Port, BaseDN: cfg.BaseDN, Security: sec},
		log: log.With().Str("component", "auth.ad").Logger(),
	}
}

func (d *adDirectory) Authenticate(ctx context.Context, username, password string) (directoryUser, error) {
	if err := ctx.Err(); err != nil {
		return directoryUser{}, err
	}
	// An empty password is an anonymous LDAP bind and would succeed.
	if username == "" || password == "" {
		return directoryUser{}, ErrInvalidCredentials
	}
	upn, err := d.ad.UPN(username)
	if err != nil {
		return directoryUser{}, d.unreachable("building the user principal name", err)
	}
	conn, err := d.ad.Connect()
	if err != nil {
		return directoryUser{}, d.unreachable("connecting", err)
	}
	defer conn.Conn.Close()

	ok, err := conn.Bind(upn, password)
	if err != nil {
		return directoryUser{}, d.unreachable("binding as the user", err)
	}
	if !ok {
		return directoryUser{}, ErrInvalidCredentials
	}

	// The person has proved who they are; nothing below may report bad credentials.
	reader, closeReader, err := d.readingConnection(conn)
	if err != nil {
		return directoryUser{}, err
	}
	defer closeReader()

	entry, err := reader.GetAttributes("userPrincipalName", upn, []string{"displayName"})
	if err != nil {
		return directoryUser{}, d.unreachable("reading the user's attributes", err)
	}
	role, err := d.role(reader, entry.DN)
	if err != nil {
		return directoryUser{}, err
	}
	if role == "" {
		d.log.Info().Str("username", username).Msg("signed in against the directory but is in no mapped group")
		return directoryUser{}, ErrNoRole
	}
	name := entry.GetAttributeValue("displayName")
	if name == "" {
		name = username
	}
	return directoryUser{Name: name, Role: role}, nil
}

// readingConnection is the connection directory reads happen on: the
// service account when bind_dn is set, else the user's own connection.
func (d *adDirectory) readingConnection(user *ldapauth.Conn) (*ldapauth.Conn, func(), error) {
	if d.cfg.BindDN == "" {
		return user, func() {}, nil
	}
	svc, err := d.ad.Connect()
	if err != nil {
		return nil, nil, d.unreachable("connecting as the service account", err)
	}
	ok, err := svc.Bind(d.cfg.BindDN, d.cfg.BindPassword)
	if err != nil || !ok {
		svc.Conn.Close()
		// A refused service account is the operator's problem, not the user's password.
		if err == nil {
			err = fmt.Errorf("bind refused for %s", d.cfg.BindDN)
		}
		return nil, nil, d.unreachable("binding as the service account", err)
	}
	return svc, func() { svc.Conn.Close() }, nil
}

// role returns the strongest role the user's groups grant, "" if none.
// A configured group missing from the directory is skipped with a warning;
// only if none resolve is the directory treated as unusable.
func (d *adDirectory) role(conn *ldapauth.Conn, userDN string) (string, error) {
	groups := make([]string, 0, len(d.cfg.GroupMap))
	for g := range d.cfg.GroupMap {
		groups = append(groups, g)
	}
	sort.Strings(groups)

	byDN := map[string]string{}
	var wanted []string
	for _, g := range groups {
		dn, err := conn.GroupDN(g)
		if err != nil {
			d.log.Warn().Str("group", g).Err(err).Msg("a group in ad.group_map is not in the directory")
			continue
		}
		byDN[strings.ToLower(dn)] = d.cfg.GroupMap[g]
		wanted = append(wanted, dn)
	}
	if len(wanted) == 0 {
		return "", d.unreachable("resolving ad.group_map", fmt.Errorf("none of the groups exist"))
	}
	matched, err := conn.ObjectGroups("dn", userDN, wanted)
	if err != nil {
		return "", d.unreachable("reading group membership", err)
	}
	role := ""
	for _, dn := range matched {
		r, ok := byDN[strings.ToLower(dn)]
		if ok && (r == RoleAdmin || role == "") {
			role = r
		}
	}
	return role, nil
}

// unreachable logs the cause server-side (never the passwords) and returns
// the sentinel callers translate to "directory unreachable".
func (d *adDirectory) unreachable(action string, err error) error {
	d.log.Error().Str("server", d.cfg.Server).Int("port", d.cfg.Port).Err(err).Msg("active directory: " + action)
	return fmt.Errorf("%w: %s: %v", ErrDirectoryUnreachable, action, err)
}
