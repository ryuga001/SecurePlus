package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	RotateOutcomeRotated   = "rotated"
	RotateOutcomeSuccessor = "successor"
	RotateOutcomeReplay    = "replay"
)

var rotateScript = redis.NewScript(`
local successor = redis.call('GET', KEYS[2])
if successor then return {'successor', successor} end
if redis.call('EXISTS', KEYS[1]) == 1 then return {'replay', ''} end
redis.call('SET', KEYS[1], '1', 'EX', ARGV[2])
redis.call('SET', KEYS[2], ARGV[1], 'EX', ARGV[3])
return {'rotated', ''}
`)

type PendingSignup struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	CodeExpires int64  `json:"code_expires"`
}

type ResetState struct {
	UserID      int    `json:"user_id"`
	Code        string `json:"code"`
	CodeExpires int64  `json:"code_expires"`
}

type Store struct {
	rdb *redis.Client
}

func NewStore(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

func Normalize(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func signupKey(email string) string      { return "auth:reg:" + email }
func signupAttempts(email string) string { return "auth:reg:" + email + ":attempts" }
func resetKey(email string) string       { return "auth:reset:" + email }
func resetAttempts(email string) string  { return "auth:reset:" + email + ":attempts" }
func blacklistKey(id string) string      { return "auth:bl:" + id }
func rotationKey(id string) string       { return "auth:rot:" + id }
func versionKey(userID int) string       { return "auth:ver:" + strconv.Itoa(userID) }
func privilegeKey(roleID int) string     { return "priv:role:" + strconv.Itoa(roleID) }
func regTokenKey(hash string) string     { return "auth:regtok:" + hash }
func identityKey(userID int) string      { return "auth:identity:" + strconv.Itoa(userID) }

func customerIdentityKey(customerID int) string {
	return "auth:identity:customer:" + strconv.Itoa(customerID)
}

func (s *Store) PutSignup(ctx context.Context, email string, pending PendingSignup, ttl time.Duration) error {
	payload, err := json.Marshal(pending)
	if err != nil {
		return err
	}

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, signupKey(email), payload, ttl)
	pipe.Del(ctx, signupAttempts(email))
	_, err = pipe.Exec(ctx)

	return err
}

func (s *Store) Signup(ctx context.Context, email string) (PendingSignup, error) {
	raw, err := s.rdb.Get(ctx, signupKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return PendingSignup{}, ErrNoPendingSignup
	}
	if err != nil {
		return PendingSignup{}, err
	}

	var pending PendingSignup
	if err := json.Unmarshal([]byte(raw), &pending); err != nil {
		return PendingSignup{}, err
	}

	return pending, nil
}

func (s *Store) DropSignup(ctx context.Context, email string) error {
	return s.rdb.Del(ctx, signupKey(email), signupAttempts(email)).Err()
}

func (s *Store) PutReset(ctx context.Context, email string, state ResetState, ttl time.Duration) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, resetKey(email), payload, ttl)
	pipe.Del(ctx, resetAttempts(email))
	_, err = pipe.Exec(ctx)

	return err
}

func (s *Store) Reset(ctx context.Context, email string) (ResetState, error) {
	raw, err := s.rdb.Get(ctx, resetKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return ResetState{}, ErrInvalidCode
	}
	if err != nil {
		return ResetState{}, err
	}

	var state ResetState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return ResetState{}, err
	}

	return state, nil
}

func (s *Store) DropReset(ctx context.Context, email string) error {
	return s.rdb.Del(ctx, resetKey(email), resetAttempts(email)).Err()
}

func (s *Store) attempt(ctx context.Context, key string, max int64, ttl time.Duration) (int64, error) {
	pipe := s.rdb.TxPipeline()
	count := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	return max - count.Val(), nil
}

func (s *Store) SignupAttempt(ctx context.Context, email string, max int64, ttl time.Duration) (int64, error) {
	return s.attempt(ctx, signupAttempts(email), max, ttl)
}

func (s *Store) ResetAttempt(ctx context.Context, email string, max int64, ttl time.Duration) (int64, error) {
	return s.attempt(ctx, resetAttempts(email), max, ttl)
}

func (s *Store) PutRegistrationToken(ctx context.Context, hash, email string, ttl time.Duration) error {
	return s.rdb.Set(ctx, regTokenKey(hash), email, ttl).Err()
}

func (s *Store) TakeRegistrationToken(ctx context.Context, hash string) (string, error) {
	email, err := s.rdb.GetDel(ctx, regTokenKey(hash)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrEmailNotVerified
	}

	return email, err
}

func (s *Store) Blacklist(ctx context.Context, id string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}

	return s.rdb.Set(ctx, blacklistKey(id), "1", ttl).Err()
}

func (s *Store) State(ctx context.Context, id string, userID int) (bool, int, error) {
	pipe := s.rdb.Pipeline()
	blocked := pipe.Exists(ctx, blacklistKey(id))
	version := pipe.Get(ctx, versionKey(userID))

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return false, 0, err
	}

	current, err := version.Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, 0, err
	}

	return blocked.Val() > 0, current, nil
}

func (s *Store) Version(ctx context.Context, userID int) (int, error) {
	value, err := s.rdb.Get(ctx, versionKey(userID)).Int()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	return value, err
}

func (s *Store) BumpVersion(ctx context.Context, userID int) error {
	return s.rdb.Incr(ctx, versionKey(userID)).Err()
}

func (s *Store) Rotate(ctx context.Context, oldID, successor string, blacklistTTL, graceTTL time.Duration) (string, string, error) {
	result, err := rotateScript.Run(ctx, s.rdb,
		[]string{blacklistKey(oldID), rotationKey(oldID)},
		successor, int(blacklistTTL.Seconds()), int(graceTTL.Seconds()),
	).Slice()
	if err != nil {
		return "", "", err
	}

	if len(result) != 2 {
		return "", "", ErrUnavailable
	}

	outcome, _ := result[0].(string)
	payload, _ := result[1].(string)

	return outcome, payload, nil
}

func (s *Store) Privileges(ctx context.Context, roleID int) ([]string, error) {
	return s.rdb.SMembers(ctx, privilegeKey(roleID)).Result()
}

func (s *Store) CachePrivileges(ctx context.Context, roleID int, names []string, ttl time.Duration) error {
	members := make([]any, 0, len(names)+1)
	for _, name := range names {
		members = append(members, name)
	}
	if len(members) == 0 {
		members = append(members, "")
	}

	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, privilegeKey(roleID))
	pipe.SAdd(ctx, privilegeKey(roleID), members...)
	pipe.Expire(ctx, privilegeKey(roleID), ttl)
	_, err := pipe.Exec(ctx)

	return err
}

func (s *Store) Identity(ctx context.Context, userID int) (IdentitySnapshot, error) {
	raw, err := s.rdb.Get(ctx, identityKey(userID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return IdentitySnapshot{}, ErrIdentityNotCached
	}
	if err != nil {
		return IdentitySnapshot{}, err
	}

	var snapshot IdentitySnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return IdentitySnapshot{}, err
	}

	return snapshot, nil
}

func (s *Store) CacheIdentity(ctx context.Context, userID int, snapshot IdentitySnapshot, ttl time.Duration) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	customerID := snapshot.Customer.ID
	if customerID == 0 {
		return s.rdb.Set(ctx, identityKey(userID), payload, ttl).Err()
	}

	index := customerIdentityKey(customerID)

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, identityKey(userID), payload, ttl)
	pipe.SAdd(ctx, index, userID)
	pipe.Expire(ctx, index, ttl)
	_, err = pipe.Exec(ctx)

	return err
}

func (s *Store) DropIdentity(ctx context.Context, userID int) error {
	return s.rdb.Del(ctx, identityKey(userID)).Err()
}

func (s *Store) DropCustomerIdentities(ctx context.Context, customerID int) error {
	index := customerIdentityKey(customerID)

	members, err := s.rdb.SMembers(ctx, index).Result()
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(members)+1)

	for _, member := range members {
		userID, err := strconv.Atoi(member)
		if err != nil {
			continue
		}

		keys = append(keys, identityKey(userID))
	}

	keys = append(keys, index)

	return s.rdb.Del(ctx, keys...).Err()
}
