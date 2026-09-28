package bot

import (
	"crypto/rand"
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"
)

// fakeShiftClient returns rewardsBefore from the first CheckRewards call, and rewardsAfter/rewardsAfterErr from every
// call after that (unless checkErr is set, which is returned from every call)
type fakeShiftClient struct {
	rewardsBefore   []shift.Reward
	rewardsAfter    []shift.Reward
	rewardsAfterErr error
	checkErr        error
	status          string
	redeemErr       error
	checkCalls      int
	redeemCalls     int
}

func (f *fakeShiftClient) CheckRewards(platform shift.Platform, game shift.Game, limit int) ([]shift.Reward, error) {
	f.checkCalls++
	if f.checkErr != nil {
		return nil, f.checkErr
	}
	if f.checkCalls == 1 {
		return f.rewardsBefore, nil
	}
	return f.rewardsAfter, f.rewardsAfterErr
}

func (f *fakeShiftClient) RedeemCode(code string, platform shift.Platform) (string, error) {
	f.redeemCalls++
	return f.status, f.redeemErr
}

// fakeStore records redemption statuses. Calling any other Store method panics (nil embedded interface)
type fakeStore struct {
	store.Store
	redemptions []string
}

func (f *fakeStore) AddRedemption(userID, code, platform string, status string) error {
	f.redemptions = append(f.redemptions, status)
	return nil
}

// panickingStore panics when getting the platform for panicUserID, and reports no platform (so they're skipped) for
// everyone else
type panickingStore struct {
	store.Store
	users       []store.UserCookies
	panicUserID string
	processed   []string
}

func (p *panickingStore) GetAllDecryptedUserCookiesSorted(limit int64) ([]store.UserCookies, error) {
	return p.users, nil
}

func (p *panickingStore) GetUserPlatformAndDM(userID string) (string, bool, error) {
	if userID == p.panicUserID {
		panic("test panic")
	}
	p.processed = append(p.processed, userID)
	return "", false, nil
}

const testCode = "AAAAA-BBBBB-CCCCC-DDDDD-EEEEE"
const testCode2 = "BBBBB-CCCCC-DDDDD-EEEEE-FFFFF"

var testUser = store.UserCookies{UserID: "123"}

// Regression: SHiFT reported success, but the rewards page didn't list any reward, which panicked with index out of range
func TestRedeemCode_SuccessWithNoRewardsListed(t *testing.T) {
	// arrange
	st := &fakeStore{}
	bot := &Bot{storage: st}
	client := &fakeShiftClient{status: shift.SUCCESS}

	// act
	reward, status, err := bot.redeemCode(client, testUser, testCode, shift.Steam)

	// assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if status != shift.SUCCESS {
		t.Errorf("Status should be %q, got %q", shift.SUCCESS, status)
	}
	if reward != nil {
		t.Errorf("Reward should be nil when no rewards are listed, got %+v", reward)
	}
	if len(st.redemptions) != 1 || st.redemptions[0] != shift.SUCCESS {
		t.Errorf("Successful redemption should still be recorded, got %v", st.redemptions)
	}
}

// The code was redeemed, so failing to look up the reward afterward shouldn't lose the redemption or count as an error
func TestRedeemCode_SuccessButRewardsCheckFails(t *testing.T) {
	// arrange
	st := &fakeStore{}
	bot := &Bot{storage: st}
	client := &fakeShiftClient{
		status:          shift.SUCCESS,
		rewardsAfterErr: errors.New("connection reset by peer"),
	}

	// act
	reward, status, err := bot.redeemCode(client, testUser, testCode, shift.Steam)

	// assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if status != shift.SUCCESS {
		t.Errorf("Status should be %q, got %q", shift.SUCCESS, status)
	}
	if reward != nil {
		t.Errorf("Reward should be nil when rewards can't be checked, got %+v", reward)
	}
	if len(st.redemptions) != 1 || st.redemptions[0] != shift.SUCCESS {
		t.Errorf("Successful redemption should still be recorded, got %v", st.redemptions)
	}
}

func TestRedeemCode_SuccessWithReward(t *testing.T) {
	// arrange
	st := &fakeStore{}
	bot := &Bot{storage: st}
	client := &fakeShiftClient{
		status:       shift.SUCCESS,
		rewardsAfter: []shift.Reward{{Title: shift.GoldenKey}},
	}

	// act
	reward, _, err := bot.redeemCode(client, testUser, testCode, shift.Steam)

	// assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if reward == nil || reward.Title != shift.GoldenKey {
		t.Errorf("Reward should be %q, got %+v", shift.GoldenKey, reward)
	}
	if len(st.redemptions) != 1 {
		t.Errorf("Redemption should be recorded, got %v", st.redemptions)
	}
}

func TestRedeemCode_ErrorButRewardsIncreased(t *testing.T) {
	// arrange
	st := &fakeStore{}
	bot := &Bot{storage: st}
	client := &fakeShiftClient{
		redeemErr:    errors.New("failed to read json text status returned from code redemption"),
		rewardsAfter: []shift.Reward{{Title: shift.GoldenKey}},
	}

	// act
	reward, status, err := bot.redeemCode(client, testUser, testCode, shift.Steam)

	// assert
	if err != nil {
		t.Fatalf("Error should be ignored when rewards increased, got %v", err)
	}
	if status != shift.SUCCESS {
		t.Errorf("Status should be %q, got %q", shift.SUCCESS, status)
	}
	if reward == nil || reward.Title != shift.GoldenKey {
		t.Errorf("Reward should be %q, got %+v", shift.GoldenKey, reward)
	}
	if len(st.redemptions) != 1 || st.redemptions[0] != shift.SUCCESS {
		t.Errorf("Redemption should be recorded as successful, got %v", st.redemptions)
	}
}

// Regression: non-final responses were recorded as redemptions, so the code was never tried again for the user
func TestRedeemCode_NonFinalResponsesAreNotRecorded(t *testing.T) {
	for _, status := range []string{shift.LINK2K, "", "Some unrecognized response"} {
		// arrange
		st := &fakeStore{}
		bot := &Bot{storage: st}
		client := &fakeShiftClient{status: status}

		// act
		_, _, err := bot.redeemCode(client, testUser, testCode, shift.Steam)

		// assert
		if err == nil {
			t.Errorf("Status %q should return an error", status)
		}
		if status == shift.LINK2K && !errors.Is(err, errLink2KAccount) {
			t.Errorf("Status %q should return errLink2KAccount, got %v", status, err)
		}
		if len(st.redemptions) != 0 {
			t.Errorf("Status %q should not be recorded as a redemption, got %v", status, st.redemptions)
		}
	}
}

func TestUserRedemptionLoop_RecoversFromPanicForOneUser(t *testing.T) {
	// arrange
	st := &panickingStore{
		users:       []store.UserCookies{{UserID: "1"}, {UserID: "2"}, {UserID: "3"}},
		panicUserID: "2",
	}
	bot := &Bot{storage: st}

	// act (a panic that escapes fails the test)
	bot.userRedemptionLoop("")

	// assert
	if len(st.processed) != 2 || st.processed[0] != "1" || st.processed[1] != "3" {
		t.Errorf("Users before and after the panicking user should still be processed, got %v", st.processed)
	}
}

// newTestBot returns a bot using a real (temporary) sqlite store, which uses client for all SHiFT requests
func newTestBot(t *testing.T, client *fakeShiftClient) (*Bot, store.Store) {
	key := make([]byte, 32)
	rand.Read(key)
	encryptor, err := store.NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.NewSqliteStore(filepath.Join(t.TempDir(), "test.db"), encryptor)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		st.Close()
	})
	bot := &Bot{
		storage: st,
		newShiftClient: func(cookies []*http.Cookie) (shiftClient, error) {
			return client, nil
		},
	}
	return bot, st
}

// addTestUser adds a logged-in user with a platform set, and DMs disabled (so no Discord session is needed)
func addTestUser(t *testing.T, st store.Store, userID string) store.UserCookies {
	cookies := []*http.Cookie{{Name: "si", Value: "si_here"}, {Name: "_session_id", Value: "session_id_here"}}
	err := st.AddUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	err = st.SetUserPlatform(userID, string(shift.Steam))
	if err != nil {
		t.Fatal(err)
	}
	err = st.EncryptAndSetUserCookies(userID, cookies)
	if err != nil {
		t.Fatal(err)
	}
	return store.UserCookies{UserID: userID, Cookies: cookies}
}

func assertAlert(t *testing.T, st store.Store, userID, expected string) {
	t.Helper()
	alert, err := st.GetUserAlert(userID)
	if err != nil {
		t.Fatal(err)
	}
	if alert != expected {
		t.Errorf("Alert should be %q, got %q", expected, alert)
	}
}

func assertUnredeemed(t *testing.T, st store.Store, userID string, expected int) {
	t.Helper()
	codes, err := st.GetValidCodesNotRedeemedForUser(userID, string(shift.Steam), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != expected {
		t.Errorf("Expected %d unredeemed codes, got %v", expected, codes)
	}
}

func assertShiftErrors(t *testing.T, st store.Store, userID string, expected int) {
	t.Helper()
	errs, err := st.GetShiftErrors(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != expected {
		t.Errorf("Expected %d shift errors, got %v", expected, errs)
	}
}

func TestRedeemCodesForUser_SessionExpired(t *testing.T) {
	// arrange
	client := &fakeShiftClient{checkErr: shift.ErrNotLoggedIn}
	bot, st := newTestBot(t, client)
	user := addTestUser(t, st, "1")
	st.AddCode(testCode, string(shift.Borderlands4), nil, nil)

	// act
	bot.redeemCodesForUser(user)

	// assert
	assertAlert(t, st, user.UserID, store.AlertSessionExpired)
	assertShiftErrors(t, st, user.UserID, 0)
	assertUnredeemed(t, st, user.UserID, 1)

	// act (until they login again, don't keep trying)
	bot.redeemCodesForUser(user)

	// assert
	if client.checkCalls != 1 {
		t.Errorf("SHiFT shouldn't be contacted again for a user with an expired session, got %d calls", client.checkCalls)
	}
}

// Regression: needing to link a 2K account was recorded as a redemption, so linking later never redeemed missed codes
func TestRedeemCodesForUser_Link2KAccountThenLinked(t *testing.T) {
	// arrange
	client := &fakeShiftClient{status: shift.LINK2K}
	bot, st := newTestBot(t, client)
	user := addTestUser(t, st, "1")
	st.AddCode(testCode, string(shift.Borderlands4), nil, nil)
	st.AddCode(testCode2, string(shift.Borderlands4), nil, nil)

	// act
	bot.redeemCodesForUser(user)

	// assert
	assertAlert(t, st, user.UserID, store.AlertLink2KAccount)
	assertShiftErrors(t, st, user.UserID, 0)
	assertUnredeemed(t, st, user.UserID, 2)
	if client.redeemCalls != 1 {
		t.Errorf("Should stop trying codes after being told to link a 2K account, got %d calls", client.redeemCalls)
	}

	// act (the user links their 2K account)
	client.status = shift.SUCCESS
	bot.redeemCodesForUser(user)

	// assert
	assertAlert(t, st, user.UserID, "")
	assertUnredeemed(t, st, user.UserID, 0)
}

func TestRedeemCodesForUser_ShiftErrorsAlert(t *testing.T) {
	// arrange
	client := &fakeShiftClient{status: shift.SUCCESS}
	bot, st := newTestBot(t, client)
	user := addTestUser(t, st, "1")
	st.AddCode(testCode, string(shift.Borderlands4), nil, nil)
	for i := 0; i < 5; i++ {
		st.AddShiftError(user.UserID, testCode, string(shift.Steam), "some error")
	}

	// act
	bot.redeemCodesForUser(user)

	// assert
	assertAlert(t, st, user.UserID, store.AlertShiftErrors)
	if client.redeemCalls != 0 {
		t.Errorf("Codes shouldn't be redeemed for users with >4 sequential errors, got %d calls", client.redeemCalls)
	}
}

func TestCheckSession_Expired(t *testing.T) {
	// arrange
	client := &fakeShiftClient{checkErr: shift.ErrNotLoggedIn}
	bot, st := newTestBot(t, client)
	user := addTestUser(t, st, "1")

	// act
	bot.checkSession(user)

	// assert
	assertAlert(t, st, user.UserID, store.AlertSessionExpired)
}

func TestCheckSession_LoggedInClearsErrors(t *testing.T) {
	// arrange
	client := &fakeShiftClient{}
	bot, st := newTestBot(t, client)
	user := addTestUser(t, st, "1")
	st.AddCode(testCode, string(shift.Borderlands4), nil, nil)
	for i := 0; i < 5; i++ {
		st.AddShiftError(user.UserID, testCode, string(shift.Steam), "some error")
	}

	// act
	bot.checkSession(user)

	// assert
	assertAlert(t, st, user.UserID, "")
	assertShiftErrors(t, st, user.UserID, 0)
}

func TestCheckSession_TemporaryErrorChangesNothing(t *testing.T) {
	// arrange
	client := &fakeShiftClient{checkErr: errors.New("invalid response code")}
	bot, st := newTestBot(t, client)
	user := addTestUser(t, st, "1")
	st.AddCode(testCode, string(shift.Borderlands4), nil, nil)
	st.AddShiftError(user.UserID, testCode, string(shift.Steam), "some error")

	// act
	bot.checkSession(user)

	// assert
	assertAlert(t, st, user.UserID, "")
	assertShiftErrors(t, st, user.UserID, 1)
}
