package bot

import (
	"errors"
	"testing"

	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"
)

// fakeShiftClient returns rewardsBefore from the first CheckRewards call, and rewardsAfter/rewardsAfterErr from every
// call after that
type fakeShiftClient struct {
	rewardsBefore   []shift.Reward
	rewardsAfter    []shift.Reward
	rewardsAfterErr error
	status          string
	redeemErr       error
	checked         bool
}

func (f *fakeShiftClient) CheckRewards(platform shift.Platform, game shift.Game, limit int) ([]shift.Reward, error) {
	if !f.checked {
		f.checked = true
		return f.rewardsBefore, nil
	}
	return f.rewardsAfter, f.rewardsAfterErr
}

func (f *fakeShiftClient) RedeemCode(code string, platform shift.Platform) (string, error) {
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
	reward, _, err := bot.redeemCode(client, testUser, testCode, shift.Steam)

	// assert
	if err != nil {
		t.Fatalf("Error should be ignored when rewards increased, got %v", err)
	}
	if reward == nil || reward.Title != shift.GoldenKey {
		t.Errorf("Reward should be %q, got %+v", shift.GoldenKey, reward)
	}
	if len(st.redemptions) != 1 {
		t.Errorf("Redemption should be recorded, got %v", st.redemptions)
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
