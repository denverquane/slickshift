package store

import (
	"crypto/rand"
	"log"
	"testing"

	"github.com/denverquane/slickshift/shift"
)

func newTestDB(t *testing.T) Store {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatal(err)
	}
	encryptor, err := NewEncryptor(key)
	if err != nil {
		log.Fatal(err)
	}
	store, err := NewSqliteStore(":memory:", encryptor)
	if err != nil {
		log.Fatal(err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return store
}

func TestSqliteStore_AddUser(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"

	if st.UserExists(userID) {
		t.Fatal("User exists when db is fresh")
	}

	err := st.AddUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.UserExists(userID) {
		t.Fatal("User does not exist after added to DB")
	}
}

func TestSqliteStore_SetUserPlatform(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	const platform = string(shift.Steam)

	st.AddUser(userID)
	p, _, err := st.GetUserPlatformAndDM(userID)
	if err != nil {
		t.Fatal(err)
	}
	if p != "" {
		t.Fatal("User platform should be empty")
	}

	err = st.SetUserPlatform(userID, platform)
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = st.GetUserPlatformAndDM(userID)
	if err != nil {
		t.Fatal(err)
	}
	if p != platform {
		t.Fatal("User platform should be " + platform)
	}
}

func TestSqliteStore_AddCode(t *testing.T) {
	st := newTestDB(t)
	const code = "AAAAA"
	const game = string(shift.Borderlands4)

	if st.CodeExists(code) {
		t.Fatal("Code exists when db is fresh")
	}

	err := st.AddCode(code, game, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !st.CodeExists(code) {
		t.Fatal("Code does not exist after added to DB")
	}
}

func TestNewSqliteStore_AddRedemptionAndGetSuccessStatus(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	const code = "AAAAA"
	const code2 = "BBBBB"
	const game = string(shift.Borderlands4)
	const platform = string(shift.Steam)
	const status = shift.SUCCESS
	const otherStatus = shift.ALREADY_REDEEMED

	st.AddUser(userID)
	st.AddCode(code, game, nil, nil)
	st.AddCode(code2, game, nil, nil)

	err := st.AddRedemption(userID, code, platform, status)
	if err != nil {
		t.Fatal(err)
	}
	err = st.AddRedemption(userID, code2, platform, otherStatus)
	if err != nil {
		t.Fatal(err)
	}

	redemptions, err := st.GetRecentRedemptionsForUser(userID, status, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(redemptions) != 1 {
		t.Fatal("Redemptions should contain 1 item")
	}
	r := redemptions[0]
	if r.Status != status {
		t.Fatal("Redemption status should be " + status)
	}
	if r.Code != code {
		t.Fatal("Redemption code should be " + code)
	}
	if r.Game != game {
		t.Fatal("Redemption game should be " + game)
	}
	if r.Platform != platform {
		t.Fatal("Redemption platform should be " + platform)
	}
	redemptions, err = st.GetRecentRedemptionsForUser(userID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(redemptions) != 2 {
		t.Fatal("Redemptions should contain 2 items when unfiltered")
	}

}

// when fetching codes for a user to redeem, if other users have marked the codes as expired or invalid, those
// codes should not be retrieved
func TestSqliteStore_GetValidCodes(t *testing.T) {
	// arrange
	st := newTestDB(t)
	const userID = "123"
	const testUserID = "234"

	const goodCode = "ABCDEF"
	const expiredCode = "BBBBB"
	const notExistCode = "CCCCC"

	const platform = string(shift.Steam)
	const game = string(shift.Borderlands4)

	st.AddUser(userID)
	st.AddUser(testUserID)

	st.SetUserPlatform(userID, platform)
	st.SetUserPlatform(testUserID, platform)

	st.AddCode(goodCode, game, nil, nil)
	st.AddCode(expiredCode, game, nil, nil)
	st.AddCode(notExistCode, game, nil, nil)

	st.AddRedemption(userID, goodCode, platform, shift.SUCCESS)
	st.AddRedemption(userID, expiredCode, platform, shift.EXPIRED)
	st.AddRedemption(userID, notExistCode, platform, shift.NOT_EXIST)

	// act
	codes, err := st.GetValidCodesNotRedeemedForUser(testUserID, platform)
	if err != nil {
		t.Fatal(err)
	}

	// assert
	if len(codes) != 1 {
		t.Fatal("Expected 1 code, got ", len(codes))
	}
	if codes[0] != goodCode {
		t.Fatal("Expected good code, got ", codes[0])
	}
}
