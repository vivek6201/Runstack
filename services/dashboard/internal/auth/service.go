package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"code-runner/dashboard/internal/config"
	"code-runner/shared/crypto"
	"code-runner/shared/models"
)

type IAuthService interface {
	Register(ctx context.Context, req RegisterRequest, meta SessionMetadata) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest, meta SessionMetadata) (*AuthResponse, error)
	ValidateSession(ctx context.Context, rawToken string) (*SessionContext, error)
	Logout(ctx context.Context, rawToken string) error
	GetUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
}

type AuthService struct {
	repo   IAuthRepository
	caches *Caches
	cfg    *config.ConfigVar
}

func NewAuthService(repo IAuthRepository, caches *Caches, cfg *config.ConfigVar) IAuthService {
	return &AuthService{
		repo:   repo,
		caches: caches,
		cfg:    cfg,
	}
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest, meta SessionMetadata) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	if _, err := s.repo.FindByEmail(ctx, email); err == nil {
		return nil, ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userID := uuid.New()
	user := models.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(hash),
		Role:         models.RoleMember,
		Plan:         models.PlanFree,
	}
	profile := buildDefaultProfile(userID, email)

	if err := s.repo.CreateWithProfile(ctx, &user, &profile); err != nil {
		return nil, err
	}

	rawToken, sessionInfo, err := s.issueSingleSession(ctx, &user, meta)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:      toUserResponse(&user, &profile),
		Session:   sessionInfo,
		Token:     rawToken,
		ExpiresAt: sessionInfo.ExpiresAt,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest, meta SessionMetadata) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	rawToken, sessionInfo, err := s.issueSingleSession(ctx, user, meta)
	if err != nil {
		return nil, err
	}

	_, profile, _ := s.repo.FindByIDWithProfile(ctx, user.ID)
	return &AuthResponse{
		User:      toUserResponse(user, profile),
		Session:   sessionInfo,
		Token:     rawToken,
		ExpiresAt: sessionInfo.ExpiresAt,
	}, nil
}

func (s *AuthService) ValidateSession(ctx context.Context, rawToken string) (*SessionContext, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, ErrSessionInvalid
	}

	sessionHash := crypto.HashAPIKey(rawToken)
	now := time.Now().UTC()

	if cached, ok := s.caches.GetSession(ctx, sessionHash); ok {
		if !cached.ExpiresAt.After(now) {
			s.caches.EvictSessionsSync(ctx, sessionHash)
			_ = s.repo.DeleteSession(ctx, sessionHash)
			return nil, ErrSessionInvalid
		}
		s.maybeTouchSession(ctx, cached, now)
		return cached, nil
	}

	sess, user, err := s.repo.FindSessionByID(ctx, sessionHash)
	if err != nil {
		return nil, ErrSessionInvalid
	}
	if !sess.ExpiresAt.After(now) {
		_ = s.repo.DeleteSession(ctx, sessionHash)
		return nil, ErrSessionInvalid
	}

	sc := buildSessionContext(sess, user)
	s.caches.SetSession(ctx, sc)
	s.maybeTouchSession(ctx, sc, now)
	return sc, nil
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return nil
	}
	sessionHash := crypto.HashAPIKey(rawToken)
	s.caches.EvictSessionsSync(ctx, sessionHash)
	return s.repo.DeleteSession(ctx, sessionHash)
}

func (s *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, profile, err := s.repo.FindByIDWithProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := toUserResponse(user, profile)
	return &resp, nil
}

func (s *AuthService) issueSingleSession(ctx context.Context, user *models.User, meta SessionMetadata) (string, *SessionInfoResponse, error) {
	rawToken, sessionHash, err := crypto.GenerateSessionToken()
	if err != nil {
		return "", nil, err
	}

	now := time.Now().UTC()
	ttl := 72 * time.Hour
	if s.cfg != nil && s.cfg.TokenTTL > 0 {
		ttl = s.cfg.TokenTTL
	}
	expiresAt := now.Add(ttl)

	provider := meta.Provider
	if provider == "" {
		provider = "password"
	}
	deviceName := meta.DeviceName
	if deviceName == "" {
		deviceName = ParseDeviceName(meta.UserAgent)
	}

	sess := &models.Session{
		ID:            sessionHash,
		UserID:        user.ID,
		Provider:      provider,
		IPAddress:     meta.IPAddress,
		UserAgent:     meta.UserAgent,
		DeviceName:    deviceName,
		ExpiresAt:     expiresAt,
		LastRotatedAt: now,
		LastActiveAt:  now,
		CreatedAt:     now,
	}

	evictedIDs, err := s.repo.CreateSingleSession(ctx, sess)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create user session: %w", err)
	}

	// Synchronously evict previous sessions in Redis so Device A cannot reuse stale cache.
	s.caches.EvictSessionsSync(ctx, evictedIDs...)
	sc := buildSessionContext(sess, user)
	s.caches.SetSession(ctx, sc)

	return rawToken, sc.ToSessionInfo(), nil
}

func (s *AuthService) maybeTouchSession(ctx context.Context, sc *SessionContext, now time.Time) {
	if now.Sub(sc.LastActiveAt) < time.Minute {
		return
	}
	sc.LastActiveAt = now
	s.caches.SetSession(ctx, sc)
	go func(hash string, ts time.Time) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.repo.TouchSession(bgCtx, hash, ts)
	}(sc.SessionID, now)
}

func buildSessionContext(sess *models.Session, user *models.User) *SessionContext {
	return &SessionContext{
		SessionID:     sess.ID,
		UserID:        user.ID,
		Email:         user.Email,
		Role:          user.Role,
		Plan:          user.Plan,
		Provider:      sess.Provider,
		IPAddress:     sess.IPAddress,
		UserAgent:     sess.UserAgent,
		DeviceName:    sess.DeviceName,
		ExpiresAt:     sess.ExpiresAt,
		LastRotatedAt: sess.LastRotatedAt,
		LastActiveAt:  sess.LastActiveAt,
		CreatedAt:     sess.CreatedAt,
	}
}

func buildDefaultProfile(userID uuid.UUID, email string) models.Profile {
	local := strings.SplitN(email, "@", 2)[0]
	cleanLocal := regexp.MustCompile(`[^a-zA-Z0-9_.-]+`).ReplaceAllString(local, "")
	if len(cleanLocal) < 3 {
		cleanLocal = "dev"
	}
	if len(cleanLocal) > 22 {
		cleanLocal = cleanLocal[:22]
	}
	suffix := strings.ReplaceAll(uuid.New().String()[:6], "-", "")
	username := strings.ToLower(cleanLocal) + "_" + suffix

	return models.Profile{
		ID:              uuid.New(),
		UserID:          userID,
		Username:        username,
		DisplayName:     local,
		DefaultLanguage: "python",
		EditorTheme:     "dark",
	}
}

func toUserResponse(user *models.User, profile *models.Profile) UserResponse {
	var profSummary *ProfileSummary
	if profile != nil {
		profSummary = &ProfileSummary{
			ID:              profile.ID,
			Username:        profile.Username,
			DisplayName:     profile.DisplayName,
			Bio:             profile.Bio,
			AvatarURL:       profile.AvatarURL,
			DefaultLanguage: profile.DefaultLanguage,
			EditorTheme:     profile.EditorTheme,
			Company:         profile.Company,
			GitHubURL:       profile.GitHubURL,
		}
	}
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Plan:      user.Plan,
		Profile:   profSummary,
		CreatedAt: user.CreatedAt,
	}
}

// ParseDeviceName extracts a concise OS + Browser label from User-Agent metadata.
func ParseDeviceName(ua string) string {
	lower := strings.ToLower(ua)
	if lower == "" {
		return "Unknown Device"
	}
	if strings.Contains(lower, "curl") {
		return "cURL CLI"
	}
	osName := detectOS(lower)
	browser := detectBrowser(lower)
	return osName + " • " + browser
}

func detectOS(lowerUA string) string {
	switch {
	case strings.Contains(lowerUA, "iphone") || strings.Contains(lowerUA, "ipad"):
		return "iOS"
	case strings.Contains(lowerUA, "android"):
		return "Android"
	case strings.Contains(lowerUA, "mac os") || strings.Contains(lowerUA, "macintosh"):
		return "macOS"
	case strings.Contains(lowerUA, "windows"):
		return "Windows"
	case strings.Contains(lowerUA, "linux"):
		return "Linux"
	default:
		return "Unknown OS"
	}
}

func detectBrowser(lowerUA string) string {
	switch {
	case strings.Contains(lowerUA, "edg/"):
		return "Edge"
	case strings.Contains(lowerUA, "chrome"):
		return "Chrome"
	case strings.Contains(lowerUA, "firefox"):
		return "Firefox"
	case strings.Contains(lowerUA, "safari"):
		return "Safari"
	default:
		return "Client"
	}
}

func validateEmail(email string) error {
	if email == "" || len(email) > 254 {
		return errors.New("valid email address is required")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || !strings.Contains(email, ".") {
		return errors.New("valid email address is required")
	}
	return nil
}

func validatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return errors.New("password cannot be blank or whitespace only")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if len([]byte(password)) > 72 {
		return errors.New("password must not exceed 72 bytes")
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("password must contain at least one letter and one number")
	}
	return nil
}
