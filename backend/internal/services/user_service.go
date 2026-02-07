package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"crypto/subtle"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"

	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/utils/pagination"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/user"
)

type Argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		memory:      64 * 1024,
		iterations:  3,
		parallelism: 2,
		saltLength:  16,
		keyLength:   32,
	}
}

type UserService struct {
	store        database.UserStore
	argon2Params *Argon2Params
}

func NewUserService(store database.UserStore) *UserService {
	return &UserService{
		store:        store,
		argon2Params: DefaultArgon2Params(),
	}
}

func (s *UserService) hashPassword(password string) (string, error) {
	salt := make([]byte, s.argon2Params.saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, s.argon2Params.iterations, s.argon2Params.memory, s.argon2Params.parallelism, s.argon2Params.keyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, s.argon2Params.memory, s.argon2Params.iterations, s.argon2Params.parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

func (s *UserService) ValidatePassword(encodedHash, password string) error {
	// Check if it's a bcrypt hash (starts with $2a$, $2b$, or $2y$)
	if strings.HasPrefix(encodedHash, "$2a$") || strings.HasPrefix(encodedHash, "$2b$") || strings.HasPrefix(encodedHash, "$2y$") {
		return s.validateBcryptPassword(encodedHash, password)
	}

	// Otherwise, assume it's Argon2
	return s.validateArgon2Password(encodedHash, password)
}

func (s *UserService) validateBcryptPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (s *UserService) validateArgon2Password(encodedHash, password string) error {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return fmt.Errorf("invalid hash format")
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return err
	}
	if version != argon2.Version {
		return fmt.Errorf("incompatible version of argon2")
	}

	var memory, iterations uint32
	var parallelism uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return err
	}

	hashLen := len(decodedHash)
	if hashLen < 0 || hashLen > 0x7fffffff {
		return fmt.Errorf("invalid hash length")
	}

	comparisonHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(hashLen))

	// constant-time compare
	if subtle.ConstantTimeCompare(comparisonHash, decodedHash) != 1 {
		return fmt.Errorf("invalid password")
	}

	return nil
}

func (s *UserService) CreateUser(ctx context.Context, user *user.ModelUser) (*user.ModelUser, error) {
	created, err := s.store.CreateUser(ctx, *user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return created, nil
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*user.ModelUser, error) {
	user, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*user.ModelUser, error) {
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) GetUserByOidcSubjectId(ctx context.Context, subjectID string) (*user.ModelUser, error) {
	user, err := s.store.GetUserByOidcSubjectID(ctx, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*user.ModelUser, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, user *user.ModelUser) (*user.ModelUser, error) {
	updated, err := s.store.SaveUser(ctx, *user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return updated, nil
}

// AttachOidcSubjectTransactional safely links an OIDC subject to the given user.
// The store implementation handles transactional locking semantics for each engine.
func (s *UserService) AttachOidcSubjectTransactional(ctx context.Context, userID string, subject string, updateFn func(u *user.ModelUser)) (*user.ModelUser, error) {
	out, err := s.store.AttachOidcSubjectTransactional(ctx, userID, subject, updateFn)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, ErrUserNotFound
	}
	return out, nil
}

func (s *UserService) CreateDefaultAdmin(ctx context.Context) error {
	// Hash password outside transaction to minimize lock time
	hashedPassword, err := s.hashPassword("arcane-admin")
	if err != nil {
		return fmt.Errorf("failed to hash default admin password: %w", err)
	}

	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}
	if count > 0 {
		slog.WarnContext(ctx, "Users already exist, skipping default admin creation")
		return nil
	}

	email := "admin@localhost"
	displayName := "Arcane Admin"
	userModel := &user.ModelUser{
		Username:               "arcane",
		Email:                  &email,
		DisplayName:            &displayName,
		PasswordHash:           hashedPassword,
		Roles:                  base.StringSlice{"admin"},
		RequiresPasswordChange: true,
	}

	if _, err := s.store.CreateUser(ctx, *userModel); err != nil {
		return fmt.Errorf("failed to create default admin user: %w", err)
	}

	slog.InfoContext(ctx, "👑 Default admin user created!")
	slog.InfoContext(ctx, "🔑 Username: arcane")
	slog.InfoContext(ctx, "🔑 Password: arcane-admin")
	slog.InfoContext(ctx, "⚠️  User will be prompted to change password on first login")

	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	_, err := s.store.DeleteUserByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (s *UserService) HashPassword(password string) (string, error) {
	return s.hashPassword(password)
}

func (s *UserService) NeedsPasswordUpgrade(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$") || strings.HasPrefix(hash, "$2y$")
}

func (s *UserService) UpgradePasswordHash(ctx context.Context, userID, password string) error {
	newHash, err := s.hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to create new hash: %w", err)
	}

	if err := s.store.UpdateUserPasswordHash(ctx, userID, newHash, time.Now().UTC()); err != nil {
		return fmt.Errorf("failed to update password hash: %w", err)
	}
	return nil
}

func (s *UserService) ListUsersPaginated(ctx context.Context, params pagination.QueryParams) ([]user.User, pagination.Response, error) {
	if params.Limit != -1 {
		if params.Limit <= 0 {
			params.Limit = 20
		} else if params.Limit > 100 {
			params.Limit = 100
		}
	}
	if params.Start < 0 {
		params.Start = 0
	}

	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, pagination.Response{}, fmt.Errorf("failed to list users: %w", err)
	}

	config := pagination.Config[user.ModelUser]{
		SearchAccessors: []pagination.SearchAccessor[user.ModelUser]{
			func(u user.ModelUser) (string, error) { return u.Username, nil },
			func(u user.ModelUser) (string, error) { return userStringPtrValue(u.Email), nil },
			func(u user.ModelUser) (string, error) { return userStringPtrValue(u.DisplayName), nil },
		},
		SortBindings: []pagination.SortBinding[user.ModelUser]{
			{Key: "username", Fn: func(a, b user.ModelUser) int { return strings.Compare(a.Username, b.Username) }},
			{Key: "displayName", Fn: func(a, b user.ModelUser) int {
				return strings.Compare(userStringPtrValue(a.DisplayName), userStringPtrValue(b.DisplayName))
			}},
			{Key: "email", Fn: func(a, b user.ModelUser) int {
				return strings.Compare(userStringPtrValue(a.Email), userStringPtrValue(b.Email))
			}},
			{Key: "lastLogin", Fn: func(a, b user.ModelUser) int { return compareOptionalTime(a.LastLogin, b.LastLogin) }},
			{Key: "createdAt", Fn: func(a, b user.ModelUser) int { return compareTime(a.CreatedAt, b.CreatedAt) }},
			{Key: "updatedAt", Fn: func(a, b user.ModelUser) int { return compareOptionalTime(a.UpdatedAt, b.UpdatedAt) }},
		},
	}

	result := pagination.SearchOrderAndPaginate(users, params, config)
	paginationResp := pagination.BuildResponseFromFilterResult(result, params)

	items := make([]user.User, 0, len(result.Items))
	for _, u := range result.Items {
		items = append(items, toUserResponseDto(u))
	}

	return items, paginationResp, nil
}

func toUserResponseDto(u user.ModelUser) user.User {
	updatedAt := u.CreatedAt
	if u.UpdatedAt != nil {
		updatedAt = *u.UpdatedAt
	}

	return user.User{
		ID:                     u.ID,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		Email:                  u.Email,
		Roles:                  u.Roles,
		OidcSubjectId:          u.OidcSubjectId,
		Locale:                 u.Locale,
		CreatedAt:              u.CreatedAt.Format("2006-01-02T15:04:05.999999Z"),
		UpdatedAt:              updatedAt.Format("2006-01-02T15:04:05.999999Z"),
		RequiresPasswordChange: u.RequiresPasswordChange,
	}
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*user.ModelUser, error) {
	slog.Debug("GetUser called", "user_id", userID)
	return s.store.GetUserByID(ctx, userID)
}

func userStringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
