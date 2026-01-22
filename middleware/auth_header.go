package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aisgo/ais-go-pkg/logger"
	"github.com/aisgo/ais-go-pkg/response"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

/* ========================================================================
 * Auth Header Spec (v1)
 * ========================================================================
 * Scope:
 *   - Gateway validates frontend token, injects user info into headers.
 *   - Downstream services verify headers and parse user info.
 *   - Service-to-service calls reuse the same scheme.
 *
 * Headers:
 *   - X-AIS-Auth-V: version ("1")
 *   - X-AIS-Auth-Iss: issuer (gateway/service name)
 *   - X-AIS-Auth-Ts: unix timestamp (seconds)
 *   - X-AIS-Auth-Nonce: random nonce
 *   - X-AIS-Auth-User: base64url(JSON UserInfo), optional for internal calls
 *   - X-AIS-Auth-Sign: hex(HMAC-SHA256(secret, payload))
 *
 * Signature payload:
 *   v|iss|ts|nonce|user
 * ======================================================================== */

const (
	AuthHeaderVersionV1 = "1"

	HeaderAuthVersion   = "X-AIS-Auth-V"
	HeaderAuthIssuer    = "X-AIS-Auth-Iss"
	HeaderAuthTimestamp = "X-AIS-Auth-Ts"
	HeaderAuthNonce     = "X-AIS-Auth-Nonce"
	HeaderAuthUser      = "X-AIS-Auth-User"
	HeaderAuthSignature = "X-AIS-Auth-Sign"
)

const (
	defaultAuthMaxAge      = 5 * time.Minute
	defaultAuthClockSkew   = 30 * time.Second
	defaultAuthNonceSize   = 16
	authContextLocalKey    = "ais_auth_ctx"
	authSignatureDelimiter = "|"
)

var (
	ErrAuthHeaderMissing          = errors.New("missing auth headers")
	ErrAuthHeaderInvalidVersion   = errors.New("invalid auth version")
	ErrAuthHeaderInvalidIssuer    = errors.New("invalid auth issuer")
	ErrAuthHeaderInvalidTS        = errors.New("invalid auth timestamp")
	ErrAuthHeaderMissingNonce     = errors.New("missing auth nonce")
	ErrAuthHeaderMissingUser      = errors.New("missing auth user")
	ErrAuthHeaderInvalidUser      = errors.New("invalid auth user header")
	ErrAuthHeaderInvalidSign      = errors.New("invalid auth signature")
	ErrAuthHeaderExpired          = errors.New("auth header expired")
	ErrAuthHeaderNotYetValid      = errors.New("auth header timestamp in future")
	ErrAuthHeaderMissingSecret    = errors.New("auth header secret is required")
	ErrAuthHeaderIssuerNotAllowed = errors.New("auth issuer not allowed")
)

// UserInfo represents the user identity injected by the gateway.
type UserInfo struct {
	UserID      string            `json:"user_id"`
	TenantID    string            `json:"tenant_id,omitempty"`
	Username    string            `json:"username,omitempty"`
	Roles       []string          `json:"roles,omitempty"`
	Permissions []string          `json:"permissions,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

// AuthContext represents verified auth header data.
type AuthContext struct {
	Version   string
	Issuer    string
	IssuedAt  time.Time
	Nonce     string
	User      *UserInfo
	UserValue string
}

// AuthHeaderValues is a structured representation of auth headers.
type AuthHeaderValues struct {
	Version   string
	Issuer    string
	Timestamp int64
	Nonce     string
	User      string
	Signature string
}

// ToMap converts AuthHeaderValues to a header map.
func (v AuthHeaderValues) ToMap() map[string]string {
	headers := map[string]string{
		HeaderAuthVersion:   v.Version,
		HeaderAuthIssuer:    v.Issuer,
		HeaderAuthTimestamp: strconv.FormatInt(v.Timestamp, 10),
		HeaderAuthNonce:     v.Nonce,
		HeaderAuthSignature: v.Signature,
	}
	if v.User != "" {
		headers[HeaderAuthUser] = v.User
	}
	return headers
}

// WriteAuthHeaders writes auth headers into http.Header.
func WriteAuthHeaders(h http.Header, v AuthHeaderValues) {
	if h == nil {
		return
	}
	if v.Version == "" && v.Issuer == "" && v.Timestamp == 0 && v.Nonce == "" && v.User == "" && v.Signature == "" {
		return
	}
	for key, value := range v.ToMap() {
		h.Set(key, value)
	}
}

// AuthContextFromContext extracts auth context from fiber.Ctx.
func AuthContextFromContext(c fiber.Ctx) (*AuthContext, bool) {
	v := c.Locals(authContextLocalKey)
	if v == nil {
		return nil, false
	}
	ctx, ok := v.(*AuthContext)
	return ctx, ok && ctx != nil
}

// UserFromContext extracts user info from fiber.Ctx.
func UserFromContext(c fiber.Ctx) (*UserInfo, bool) {
	ctx, ok := AuthContextFromContext(c)
	if !ok || ctx.User == nil {
		return nil, false
	}
	return ctx.User, true
}

// AuthHeaderSignerConfig configures header signing.
type AuthHeaderSignerConfig struct {
	Enabled bool   `yaml:"enabled"`
	Secret  string `yaml:"secret"`
	Issuer  string `yaml:"issuer"`
	Version string `yaml:"version"`

	NowFunc func() time.Time `yaml:"-"`
}

// AuthHeaderSigner signs auth headers for gateway/service-to-service calls.
type AuthHeaderSigner struct {
	config  AuthHeaderSignerConfig
	nowFunc func() time.Time
}

// NewAuthHeaderSigner creates a new signer.
func NewAuthHeaderSigner(cfg *AuthHeaderSignerConfig) *AuthHeaderSigner {
	if cfg == nil {
		cfg = &AuthHeaderSignerConfig{}
	}
	config := *cfg
	if config.Version == "" {
		config.Version = AuthHeaderVersionV1
	}
	signer := &AuthHeaderSigner{config: config}
	if config.NowFunc != nil {
		signer.nowFunc = config.NowFunc
	} else {
		signer.nowFunc = time.Now
	}
	return signer
}

// BuildHeaders builds auth headers for the given user.
func (s *AuthHeaderSigner) BuildHeaders(user *UserInfo) (AuthHeaderValues, error) {
	if !s.config.Enabled {
		return AuthHeaderValues{}, nil
	}
	if s.config.Secret == "" {
		return AuthHeaderValues{}, ErrAuthHeaderMissingSecret
	}
	if s.config.Issuer == "" {
		return AuthHeaderValues{}, ErrAuthHeaderInvalidIssuer
	}
	userValue, err := EncodeUserInfo(user)
	if err != nil {
		return AuthHeaderValues{}, err
	}
	nonce, err := generateNonce()
	if err != nil {
		return AuthHeaderValues{}, err
	}
	issuedAt := s.nowFunc().Unix()
	signature := signAuthHeader(s.config.Secret, s.config.Version, s.config.Issuer, issuedAt, nonce, userValue)
	return AuthHeaderValues{
		Version:   s.config.Version,
		Issuer:    s.config.Issuer,
		Timestamp: issuedAt,
		Nonce:     nonce,
		User:      userValue,
		Signature: signature,
	}, nil
}

// AuthHeaderVerifierConfig configures header verification.
type AuthHeaderVerifierConfig struct {
	Enabled           bool              `yaml:"enabled"`
	Secret            string            `yaml:"secret"`
	Secrets           map[string]string `yaml:"secrets"`
	AllowedIssuers    []string          `yaml:"allowed_issuers"`
	Version           string            `yaml:"version"`
	MaxAge            time.Duration     `yaml:"max_age"`
	AllowedClockSkew  time.Duration     `yaml:"allowed_clock_skew"`
	AllowEmptyUser    bool              `yaml:"allow_empty_user"`
	AllowMissingNonce bool              `yaml:"allow_missing_nonce"`

	NowFunc func() time.Time `yaml:"-"`
}

// AuthHeaderVerifier verifies headers and injects auth context.
type AuthHeaderVerifier struct {
	config  AuthHeaderVerifierConfig
	log     *logger.Logger
	nowFunc func() time.Time
}

// NewAuthHeaderVerifier creates a verifier.
func NewAuthHeaderVerifier(cfg *AuthHeaderVerifierConfig, log *logger.Logger) *AuthHeaderVerifier {
	if cfg == nil {
		cfg = &AuthHeaderVerifierConfig{}
	}
	config := *cfg
	if config.Version == "" {
		config.Version = AuthHeaderVersionV1
	}
	if config.MaxAge == 0 {
		config.MaxAge = defaultAuthMaxAge
	}
	if config.AllowedClockSkew == 0 {
		config.AllowedClockSkew = defaultAuthClockSkew
	}
	if log == nil {
		log = logger.NewNop()
	}
	verifier := &AuthHeaderVerifier{config: config, log: log}
	if config.NowFunc != nil {
		verifier.nowFunc = config.NowFunc
	} else {
		verifier.nowFunc = time.Now
	}
	return verifier
}

// Authenticate returns a Fiber middleware for auth header verification.
func (v *AuthHeaderVerifier) Authenticate() fiber.Handler {
	return func(c fiber.Ctx) error {
		if !v.config.Enabled {
			return c.Next()
		}
		if v.config.Secret == "" && len(v.config.Secrets) == 0 {
			v.log.Error("Auth header verifier misconfigured: missing secret")
			return response.InternalError(c, "auth header misconfigured")
		}
		values, err := ParseAuthHeaderValuesFromFiber(c)
		if err != nil {
			v.log.Warn("Auth header parse failed",
				zap.Error(err),
				zap.String("path", c.Path()),
				zap.String("ip", c.IP()),
			)
			return response.Unauthorized(c, err.Error())
		}
		ctx, err := v.Verify(values)
		if err != nil {
			v.log.Warn("Auth header verify failed",
				zap.Error(err),
				zap.String("issuer", values.Issuer),
				zap.String("path", c.Path()),
				zap.String("ip", c.IP()),
			)
			return response.Unauthorized(c, err.Error())
		}
		c.Locals(authContextLocalKey, ctx)
		return c.Next()
	}
}

// Verify verifies auth header values and returns auth context.
func (v *AuthHeaderVerifier) Verify(values AuthHeaderValues) (*AuthContext, error) {
	if values.Version == "" || values.Issuer == "" || values.Timestamp == 0 || values.Signature == "" {
		return nil, ErrAuthHeaderMissing
	}
	if v.config.Version != "" && values.Version != v.config.Version {
		return nil, ErrAuthHeaderInvalidVersion
	}
	if !v.isIssuerAllowed(values.Issuer) {
		return nil, ErrAuthHeaderIssuerNotAllowed
	}
	if !v.config.AllowMissingNonce && values.Nonce == "" {
		return nil, ErrAuthHeaderMissingNonce
	}
	secret := v.secretForIssuer(values.Issuer)
	if secret == "" {
		return nil, ErrAuthHeaderMissingSecret
	}
	expected := signAuthHeader(secret, values.Version, values.Issuer, values.Timestamp, values.Nonce, values.User)
	if !secureCompare(expected, values.Signature) {
		return nil, ErrAuthHeaderInvalidSign
	}
	issuedAt := time.Unix(values.Timestamp, 0)
	now := v.nowFunc()
	if v.config.MaxAge > 0 && now.Sub(issuedAt) > v.config.MaxAge {
		return nil, ErrAuthHeaderExpired
	}
	if issuedAt.After(now.Add(v.config.AllowedClockSkew)) {
		return nil, ErrAuthHeaderNotYetValid
	}
	user, err := DecodeUserInfo(values.User)
	if err != nil {
		return nil, ErrAuthHeaderInvalidUser
	}
	if !v.config.AllowEmptyUser {
		if user == nil || user.UserID == "" {
			return nil, ErrAuthHeaderMissingUser
		}
	}
	return &AuthContext{
		Version:   values.Version,
		Issuer:    values.Issuer,
		IssuedAt:  issuedAt,
		Nonce:     values.Nonce,
		User:      user,
		UserValue: values.User,
	}, nil
}

// ParseAuthHeaderValuesFromFiber reads auth headers from fiber.Ctx.
func ParseAuthHeaderValuesFromFiber(c fiber.Ctx) (AuthHeaderValues, error) {
	return parseAuthHeaderValues(func(key string) string { return c.Get(key) })
}

// ParseAuthHeaderValuesFromHeader reads auth headers from http.Header.
func ParseAuthHeaderValuesFromHeader(h http.Header) (AuthHeaderValues, error) {
	if h == nil {
		return AuthHeaderValues{}, ErrAuthHeaderMissing
	}
	return parseAuthHeaderValues(h.Get)
}

func parseAuthHeaderValues(get func(string) string) (AuthHeaderValues, error) {
	version := strings.TrimSpace(get(HeaderAuthVersion))
	issuer := strings.TrimSpace(get(HeaderAuthIssuer))
	stamp := strings.TrimSpace(get(HeaderAuthTimestamp))
	signature := strings.TrimSpace(get(HeaderAuthSignature))
	if version == "" || issuer == "" || stamp == "" || signature == "" {
		return AuthHeaderValues{}, ErrAuthHeaderMissing
	}
	timestamp, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil || timestamp <= 0 {
		return AuthHeaderValues{}, ErrAuthHeaderInvalidTS
	}
	return AuthHeaderValues{
		Version:   version,
		Issuer:    issuer,
		Timestamp: timestamp,
		Nonce:     strings.TrimSpace(get(HeaderAuthNonce)),
		User:      strings.TrimSpace(get(HeaderAuthUser)),
		Signature: signature,
	}, nil
}

func (v *AuthHeaderVerifier) secretForIssuer(issuer string) string {
	if issuer == "" {
		return ""
	}
	if len(v.config.Secrets) > 0 {
		if secret, ok := v.config.Secrets[issuer]; ok {
			return secret
		}
		if v.config.Secret != "" {
			return v.config.Secret
		}
		return ""
	}
	return v.config.Secret
}

func (v *AuthHeaderVerifier) isIssuerAllowed(issuer string) bool {
	if issuer == "" {
		return false
	}
	if len(v.config.AllowedIssuers) == 0 {
		return true
	}
	for _, allowed := range v.config.AllowedIssuers {
		if issuer == allowed {
			return true
		}
	}
	return false
}

// EncodeUserInfo encodes user info into base64url JSON.
func EncodeUserInfo(user *UserInfo) (string, error) {
	if user == nil {
		return "", nil
	}
	data, err := json.Marshal(user)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// DecodeUserInfo decodes base64url JSON into user info.
func DecodeUserInfo(value string) (*UserInfo, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		data, err = base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, err
		}
	}
	var user UserInfo
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func signAuthHeader(secret, version, issuer string, timestamp int64, nonce, user string) string {
	payload := buildSignaturePayload(version, issuer, timestamp, nonce, user)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func buildSignaturePayload(version, issuer string, timestamp int64, nonce, user string) string {
	parts := []string{
		version,
		issuer,
		strconv.FormatInt(timestamp, 10),
		nonce,
		user,
	}
	return strings.Join(parts, authSignatureDelimiter)
}

func secureCompare(expected, provided string) bool {
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func generateNonce() (string, error) {
	buf := make([]byte, defaultAuthNonceSize)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
